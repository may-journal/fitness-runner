#!/usr/bin/env node
/**
 * Discover @mayjournal publishable workspaces; seed missing packages on npm;
 * configure trusted publishing for publish.yml.
 */
import { spawnSync } from 'node:child_process';
import { existsSync, renameSync } from 'node:fs';
import { join, dirname } from 'node:path';
import { fileURLToPath } from 'node:url';
import { listPublishablePackages } from '../list-publishable-packages/index.mjs';

const root = join(dirname(fileURLToPath(import.meta.url)), '../..');
const repo = process.env.NPM_TRUST_REPO ?? 'may-journal/fitness-runner';
const workflow = process.env.NPM_TRUST_WORKFLOW ?? 'publish.yml';
const minNpm = '11.10.0';
const dryRun = Boolean(process.env.DRY_RUN);
const npmrcBackup = join(root, '.npmrc.setup-trust.bak');

/** @type {'all' | 'seed' | 'trust' | 'check'} */
const mode = parseMode(process.argv[2]);

let failed = false;
let npmrcMoved = false;

function parseMode(arg) {
  const normalized = (arg ?? 'all').replace(/^--/, '');
  if (['all', 'seed', 'trust', 'check'].includes(normalized)) return normalized;
  console.error('Usage: npm run provision:npm [-- check|seed|trust]');
  process.exit(2);
}

function restoreNpmrc() {
  if (npmrcMoved && existsSync(npmrcBackup)) {
    renameSync(npmrcBackup, join(root, '.npmrc'));
    npmrcMoved = false;
  }
}

function prepareLocalAuth() {
  const hasToken = Boolean(process.env.NODE_AUTH_TOKEN);
  const npmrcPath = join(root, '.npmrc');
  if (!hasToken && existsSync(npmrcPath)) {
    renameSync(npmrcPath, npmrcBackup);
    npmrcMoved = true;
    process.on('exit', restoreNpmrc);
  }
}

function npmVersionOk() {
  const result = spawnSync('npm', ['--version'], { encoding: 'utf8' });
  const v = (result.stdout ?? '0').trim().split('.').map(Number);
  const m = minNpm.split('.').map(Number);
  return (
    v[0] > m[0] ||
    (v[0] === m[0] && v[1] > m[1]) ||
    (v[0] === m[0] && v[1] === m[1] && (v[2] ?? 0) >= (m[2] ?? 0))
  );
}

/** @param {string[]} args */
function npmCmd(args, options = {}) {
  const bin = process.env.NPM_CLI
    ? [process.env.NPM_CLI]
    : npmVersionOk()
      ? ['npm']
      : ['npx', '--yes', `npm@${minNpm}`];
  const result = spawnSync(bin[0], [...bin.slice(1), ...args], {
    cwd: options.cwd ?? root,
    encoding: 'utf8',
    stdio: options.stdio ?? 'pipe',
  });
  return result;
}

function requireLogin() {
  if (!process.env.NODE_AUTH_TOKEN) {
    const whoami = npmCmd(['whoami']);
    if (whoami.status !== 0) {
      console.error('Not logged in. Set NPM_PROVISION_TOKEN (CI) or run: npm login');
      process.exit(1);
    }
    console.log(`npm user: ${(whoami.stdout ?? '').trim()}`);
  } else {
    const whoami = npmCmd(['whoami']);
    if (whoami.status === 0) {
      console.log(`npm user: ${(whoami.stdout ?? '').trim()}`);
    } else {
      console.log('npm auth: NPM_PROVISION_TOKEN / NODE_AUTH_TOKEN');
    }
  }
}

/** @param {string} name */
function packageExists(name) {
  return npmCmd(['view', name, 'version']).status === 0;
}

/** @param {string} name */
function trustConfigured(name) {
  const result = npmCmd(['trust', 'list', name, '--json']);
  if (result.status !== 0) return false;
  const text = (result.stdout ?? '').trim();
  if (!text) return false;
  try {
    const configs = JSON.parse(text);
    return Array.isArray(configs) && configs.length > 0;
  } catch {
    return false;
  }
}

/** @param {{ dir: string, name: string }} pkg */
function seedPackage(pkg) {
  if (packageExists(pkg.name)) {
    console.log(`  registry: ${pkg.name} already exists`);
    return true;
  }
  console.log(`  registry: publishing ${pkg.name} (first time) from ${pkg.dir}`);
  if (dryRun) {
    console.log('  [dry-run] would npm publish --access restricted');
    return true;
  }
  const result = npmCmd(['publish', '--access', 'restricted'], {
    cwd: join(root, pkg.dir),
    stdio: 'inherit',
  });
  return result.status === 0;
}

/** @param {string} name */
function configureTrust(name) {
  if (trustConfigured(name)) {
    console.log(`  trust: ${name} already configured`);
    return true;
  }
  console.log(`  trust: ${name} → GitHub ${repo} / ${workflow}`);
  if (dryRun) {
    console.log('  [dry-run] would npm trust github');
    return true;
  }
  const result = npmCmd(['trust', 'github', name, '--file', workflow, '--repo', repo, '--yes'], {
    stdio: 'inherit',
  });
  return result.status === 0;
}

function sleep(ms) {
  return new Promise((resolve) => setTimeout(resolve, ms));
}

async function runCheck() {
  requireLogin();
  const packages = listPublishablePackages();
  const missingRegistry = [];
  const missingTrust = [];

  for (const pkg of packages) {
    if (!packageExists(pkg.name)) {
      missingRegistry.push(pkg.name);
      continue;
    }
    if (!trustConfigured(pkg.name)) {
      missingTrust.push(pkg.name);
    }
  }

  if (missingRegistry.length === 0 && missingTrust.length === 0) {
    console.log('All publishable packages exist on npm with trusted publishing configured.');
    return;
  }

  if (missingRegistry.length > 0) {
    console.error('Missing on npm registry:');
    for (const name of missingRegistry) console.error(`  - ${name}`);
  }
  if (missingTrust.length > 0) {
    console.error(`Missing trusted publisher (Publish workflow / ${workflow}):`);
    for (const name of missingTrust) console.error(`  - ${name}`);
  }
  console.error('');
  console.error('Run the Provision npm packages workflow or: npm run provision:npm');
  process.exit(1);
}

async function runProvision() {
  const doSeed = mode === 'all' || mode === 'seed';
  const doTrust = mode === 'all' || mode === 'trust';

  console.log(`=== provision npm packages (mode=${mode}) ===`);
  console.log(`repo: ${repo}  workflow: ${workflow}`);
  if (!process.env.NODE_AUTH_TOKEN && doTrust) {
    console.log('Local trust setup needs npm 2FA in the browser (use npm 5-minute skip for bulk).');
  }
  console.log('');
  requireLogin();

  const packages = listPublishablePackages();

  if (doSeed) {
    console.log('=== build ===');
    if (dryRun) {
      console.log('[dry-run] would npm run build');
    } else {
      const build = npmCmd(['run', 'build'], { stdio: 'inherit' });
      if (build.status !== 0) process.exit(build.status ?? 1);
    }
    console.log('');
    console.log('=== seed registry ===');
    for (const pkg of packages) {
      console.log(pkg.name);
      if (!seedPackage(pkg)) {
        console.error(`  WARN: seed failed for ${pkg.name}`);
        failed = true;
      }
    }
  }

  if (doTrust) {
    console.log('');
    console.log('=== trusted publishing ===');
    for (const pkg of packages) {
      console.log(pkg.name);
      if (!packageExists(pkg.name)) {
        console.error('  SKIP: not on registry yet');
        continue;
      }
      if (!configureTrust(pkg.name)) {
        console.error(`  WARN: trust failed for ${pkg.name}`);
        failed = true;
      }
      await sleep(2000);
    }
  }

  if (failed) {
    console.error('=== finished with errors ===');
    process.exit(1);
  }
  console.log('=== done ===');
}

prepareLocalAuth();

if (mode === 'check') {
  await runCheck();
} else {
  await runProvision();
}

restoreNpmrc();

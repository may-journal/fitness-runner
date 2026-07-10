#!/usr/bin/env node
/**
 * Discover @mayjournal publishable workspaces; seed missing packages on npm;
 * configure trusted publishing for publish.yml.
 */
import { spawnSync as realSpawnSync } from 'node:child_process';
import { existsSync, renameSync } from 'node:fs';
import { homedir } from 'node:os';
import { join, dirname } from 'node:path';
import { fileURLToPath } from 'node:url';
import { listPublishablePackages } from '../list-publishable-packages/index.mjs';
import { TRUST, configureTrustWithPermissions } from './trust-github-api.mjs';
import {
  JSON_FLAG,
  NPM,
  NPMRC,
  NPM_SCRIPT_BUILD,
  REPO_ROOT,
  RUN_SUBCOMMAND,
} from '../constants.cjs';

/** The provision-everything mode (seed + trust). */
const ALL = 'all';

const root = REPO_ROOT;
const repo = process.env.NPM_TRUST_REPO ?? 'may-journal/fitness-runner';
const workflow = process.env.NPM_TRUST_WORKFLOW ?? 'publish.yml';
const minNpm = '11.14.1';
const dryRun = Boolean(process.env.DRY_RUN);
const npmrcBackup = join(root, '.npmrc.setup-trust.bak');

/** @type {{ spawnSync?: typeof realSpawnSync, configureTrustWithPermissions?: typeof configureTrustWithPermissions }} */
export const testHooks = {};

let failed = false;
let npmrcMoved = false;

export function parseMode(arg) {
  const normalized = (arg ?? ALL).replace(/^--/, '');
  if ([ALL, 'seed', TRUST, 'check'].includes(normalized)) return normalized;
  console.error('Usage: npm run provision:npm [-- check|seed|trust]');
  process.exit(2);
}

function restoreNpmrc() {
  if (npmrcMoved && existsSync(npmrcBackup)) {
    renameSync(npmrcBackup, join(root, NPMRC));
    npmrcMoved = false;
  }
}

/** npmrc paths for trust API after repo .npmrc was moved aside. */
export function getTrustAuthNpmrcPaths() {
  return npmrcMoved ? [join(homedir(), NPMRC)] : undefined;
}

function prepareLocalAuth() {
  const hasToken = Boolean(process.env.NODE_AUTH_TOKEN);
  const npmrcPath = join(root, NPMRC);
  if (!hasToken && existsSync(npmrcPath)) {
    renameSync(npmrcPath, npmrcBackup);
    npmrcMoved = true;
    process.on('exit', restoreNpmrc);
  }
}

function npmVersionOk() {
  const spawnSync = testHooks.spawnSync ?? realSpawnSync;
  const result = spawnSync(NPM, ['--version'], { encoding: 'utf8' });
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
      ? [NPM]
      : ['npx', '--yes', `npm@${minNpm}`];
  const spawnSync = testHooks.spawnSync ?? realSpawnSync;
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
  const result = npmCmd([TRUST, 'list', name, JSON_FLAG]);
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
    console.log('  [dry-run] would npm publish --access public');
    return true;
  }
  const result = npmCmd(['publish', '--access', 'public'], {
    cwd: join(root, pkg.dir),
    stdio: 'inherit',
  });
  return result.status === 0;
}

/** @param {string} name */
export async function configureTrust(name) {
  if (trustConfigured(name)) {
    console.log(`  trust: ${name} already configured`);
    return true;
  }
  console.log(`  trust: ${name} → GitHub ${repo} / ${workflow}`);
  if (Boolean(process.env.DRY_RUN)) {
    console.log('  [dry-run] would configure trusted publisher (createPackage)');
    return true;
  }
  if (process.env.NODE_AUTH_TOKEN) {
    console.error(
      '  SKIP: trusted publishing cannot be set with NPM_PROVISION_TOKEN (needs interactive 2FA).'
    );
    console.error('  Run locally: npm run provision:npm -- trust');
    return false;
  }
  if (!testHooks.configureTrustWithPermissions) {
    console.log('  trying: npm trust github (browser 2FA — same as npm login)');
    const trustArgs = [TRUST, 'github', name, '--file', workflow, '--repo', repo, '--yes'];
    const cli = npmCmd(trustArgs, { stdio: 'inherit' });
    if (cli.status === 0) {
      console.log(`  trust: ${name} configured via npm CLI`);
      return true;
    }
    console.log(
      '  npm trust github did not succeed; trying registry API with createPackage permission…'
    );
  }

  try {
    const configureApi = testHooks.configureTrustWithPermissions ?? configureTrustWithPermissions;
    await configureApi({
      npmrcPaths: getTrustAuthNpmrcPaths(),
      packageName: name,
      repo,
      workflow,
    });
    return true;
  } catch (err) {
    const code = err.statusCode ?? err.code;
    if (code === 409) {
      console.log(`  trust: ${name} already configured`);
      return true;
    }
    if (code === 400) {
      console.error(
        '  Hint: registry requires permissions in trust POST; use npm 11.14+ and complete browser 2FA.'
      );
    }
    if (code === 401) {
      console.error(
        '  Hint: run in a terminal, complete browser sign-in when prompted, or configure trust at https://www.npmjs.com/package/' +
          name.replace('/', '%2f') +
          '/settings'
      );
    }
    console.error(`  ${err.message ?? err}`);
    return false;
  }
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
  console.error('Configure trusted publishing on npmjs.com or run: npm run provision:npm -- trust');
  process.exit(1);
}

/** @param {'all' | 'seed' | 'trust' | 'check'} mode */
async function runProvision(mode) {
  const doSeed = mode === ALL || mode === 'seed';
  const doTrust = mode === ALL || mode === TRUST;

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
      const build = npmCmd([RUN_SUBCOMMAND, NPM_SCRIPT_BUILD], { stdio: 'inherit' });
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
      if (!(await configureTrust(pkg.name))) {
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

const isMain = process.argv[1] === fileURLToPath(import.meta.url);

if (isMain) {
  const mode = parseMode(process.argv[2]);
  prepareLocalAuth();

  if (mode === 'check') {
    await runCheck();
  } else {
    await runProvision(mode);
  }

  restoreNpmrc();
}

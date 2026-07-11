#!/usr/bin/env node
/**
 * POST trusted publisher config with required permissions (registry API since 2026-05-20).
 * npm trust github omits permissions and returns E400 until npm/cli is updated.
 */
import { existsSync, readdirSync, readFileSync } from 'node:fs';
import { homedir } from 'node:os';
import { join, dirname } from 'node:path';
import { createRequire } from 'node:module';
import { fileURLToPath } from 'node:url';
import readline from 'node:readline/promises';
import { stdin as input, stdout as output } from 'node:process';
import { NODE_MODULES, NPMRC, REPO_ROOT } from '../constants.cjs';

/** The npm `trust` subcommand — shared with the provisioning entry point. */
export const TRUST = 'trust';

const minNpm = '11.14.1';

function compareVer(a, b) {
  const pa = a.split('.').map(Number);
  const pb = b.split('.').map(Number);
  for (let i = 0; i < 3; i++) {
    if ((pa[i] ?? 0) !== (pb[i] ?? 0)) return (pa[i] ?? 0) - (pb[i] ?? 0);
  }
  return 0;
}

/** Load npm-registry-fetch + webAuthOpener from an npx npm install. */
function loadNpmDeps() {
  const npxBase = join(homedir(), '.npm/_npx');
  if (!existsSync(npxBase)) {
    throw new Error(`No npx cache at ${npxBase}. Run: npx --yes npm@${minNpm} -v`);
  }
  let best = null;
  for (const entry of readdirSync(npxBase)) {
    const npmPkg = join(npxBase, entry, 'node_modules/npm/package.json');
    if (!existsSync(npmPkg)) continue;
    const ver = JSON.parse(readFileSync(npmPkg, 'utf8')).version;
    if (compareVer(ver, minNpm) < 0) continue;
    if (!best || compareVer(ver, best.ver) > 0)
      best = { dir: join(npxBase, entry, NODE_MODULES), ver };
  }
  if (!best) {
    throw new Error(`Install npm ${minNpm}+ via npx first: npx --yes npm@${minNpm} -v`);
  }
  const require = createRequire(join(best.dir, 'npm/package.json'));
  const profile = require('npm-profile');
  return {
    loginWeb: profile.loginWeb,
    npa: require('npm-package-arg'),
    npmFetch: require('npm-registry-fetch'),
    webAuthOpener: profile.webAuthOpener,
  };
}

/** @param {unknown} body */
function otpChallengeBody(body) {
  if (body && typeof body === 'object' && !Array.isArray(body)) return body;
  if (typeof body === 'string') {
    try {
      const parsed = JSON.parse(body);
      if (parsed && typeof parsed === 'object' && !Array.isArray(parsed)) return parsed;
    } catch {
      /* ignore */
    }
  }
  return null;
}

/** Registry web 2FA (authUrl + doneUrl) may appear with EOTP or plain E401. */
export function isWebOtpChallenge(err) {
  const body = otpChallengeBody(err?.body);
  return Boolean(body?.authUrl && body?.doneUrl);
}

/** @param {Error & { uri?: string, message?: string }} err */
function isTrustRequest(err) {
  const target = String(err.uri ?? err.message ?? '');
  return /\/trust$/i.test(target);
}

/** Needs browser sign-in (passkey): EOTP, embedded web URLs, or plain 401 on trust POST. */
export function needsWebSignIn(err) {
  if (isWebOtpChallenge(err)) return true;
  if (err?.code === 'EOTP') return true;
  if ((err?.statusCode === 401 || err?.code === 'E401') && isTrustRequest(err)) return true;
  if (err?.code === 'E401' && err.statusCode === 401) {
    const raw = typeof err.body === 'string' ? err.body : JSON.stringify(err.body ?? '');
    if (/one-time pass/i.test(raw)) return true;
  }
  return false;
}

/** @param {string[] | undefined} pathsOverride */
export function loadAuthFromNpmrc(pathsOverride) {
  const paths = pathsOverride ?? [join(homedir(), NPMRC), join(REPO_ROOT, NPMRC)];
  for (const p of paths) {
    if (!existsSync(p)) continue;
    const text = readFileSync(p, 'utf8');
    const m = text.match(/\/\/registry\.npmjs\.org\/:_authToken=(\S+)/);
    if (m?.[1] && !m[1].includes('${')) return m[1];
  }
  return process.env.NODE_AUTH_TOKEN ?? null;
}

async function openBrowserHint(url) {
  output.write(
    `\nSign in at (passkey works; enable “skip 2FA for 5 minutes” on npm for bulk trust setup):\n${url}\n`
  );
  const rl = readline.createInterface({ input, output });
  await rl.question('Press ENTER to open in the browser… ');
  rl.close();
  if (process.platform === 'darwin') {
    const { spawn } = await import('node:child_process');
    spawn('open', [url], { detached: true, stdio: 'ignore' }).unref();
  }
}

/**
 * Trust auth is two steps: refresh Bearer token (npm login web), then operation OTP
 * (authUrl/doneUrl). loginWeb returns a session token — not an npm-otp value.
 *
 * @param {typeof loadNpmDeps} [depsLoader]
 * @param {{ sessionRefreshed?: boolean }} [state]
 */
export async function otplease(opts, fn, depsLoader = loadNpmDeps, state = {}) {
  try {
    return await fn(opts);
  } catch (err) {
    if (!input.isTTY || !output.isTTY) {
      throw err;
    }

    if (isWebOtpChallenge(err)) {
      const body = otpChallengeBody(err.body);
      const { webAuthOpener } = depsLoader();
      output.write(
        '\nApprove this trust change in the browser (passkey OK). Wait until npm confirms before returning.\n'
      );
      const { token: otp } = await webAuthOpener(
        (url) => openBrowserHint(url),
        body.authUrl,
        body.doneUrl,
        opts
      );
      return await fn({ ...opts, otp });
    }

    if (!state.sessionRefreshed && needsWebSignIn(err)) {
      const { loginWeb } = depsLoader();
      output.write(
        '\nSign in to npm in the browser (passkey OK). Finish on npmjs.com, then return here.\n'
      );
      const { token } = await loginWeb((url) => openBrowserHint(url), {
        ...opts,
        authType: 'web',
      });
      return otplease({ ...opts, token }, fn, depsLoader, { sessionRefreshed: true });
    }

    throw err;
  }
}

/** @param {string} repo @param {string} workflow */
export function trustPostBody(repo, workflow) {
  return [
    {
      claims: {
        repository: repo,
        workflow_ref: { file: workflow },
      },
      permissions: ['createPackage'],
      type: 'github',
    },
  ];
}

/**
 * @param {{ packageName: string, repo: string, workflow: string, registry?: string, npmrcPaths?: string[] }} opts
 * @param {{ npmFetch: Function, npa: Function } | undefined} depsOverride
 */
export async function configureTrustWithPermissions(opts, depsOverride) {
  const {
    packageName,
    repo,
    workflow,
    registry = 'https://registry.npmjs.org/',
    npmrcPaths,
  } = opts;
  const token = loadAuthFromNpmrc(npmrcPaths);
  if (!token) {
    throw new Error('Not logged in. Run npm login or set NODE_AUTH_TOKEN.');
  }

  const { npmFetch, npa } = depsOverride ?? loadNpmDeps();
  const spec = npa(packageName);
  const uri = `/-/package/${spec.escapedName}/trust`;
  const body = trustPostBody(repo, workflow);

  const baseOpts = {
    authType: 'web',
    npmCommand: TRUST,
    registry,
    token,
  };

  await otplease(baseOpts, (reqOpts) => npmFetch(uri, { ...reqOpts, body, method: 'POST' }));
  return true;
}

if (process.argv[1] === fileURLToPath(import.meta.url)) {
  const [, , packageName, repo, workflow] = process.argv;
  if (!packageName || !repo || !workflow) {
    console.error('Usage: trust-github-api.mjs <package> <owner/repo> <workflow.yml>');
    process.exit(2);
  }
  try {
    await configureTrustWithPermissions({ packageName, repo, workflow });
  } catch (err) {
    if (err.statusCode === 409) {
      console.log('Trust already configured.');
      process.exit(0);
    }
    console.error(err.message ?? err);
    process.exit(1);
  }
}

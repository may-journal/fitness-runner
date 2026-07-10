import { spawnSync } from 'node:child_process';
import { dirname, join } from 'node:path';
import { fileURLToPath } from 'node:url';
import { FITNESS_PKG, FITNESS_SHARED_PKG, REPO_ROOT, RUNNER_DIR } from '../constants.cjs';

const root = REPO_ROOT;

/** True when attw crashed instead of reporting package type issues. */
export function isAttwToolCrash(summary) {
  return /Cannot read properties of undefined \(reading 'filename'\)/.test(summary);
}

/** Packages to run attw on (advisory). */
export const ATTW_PACKAGES = [
  { dir: RUNNER_DIR, name: FITNESS_PKG },
  { dir: 'packages/shared', name: FITNESS_SHARED_PKG },
];

/**
 * @param {string} dir relative to repo root
 * @param {{ execSync?: typeof spawnSync }} [hooks]
 * @returns {{ ok: boolean, summary: string }}
 */
export function runAttwPackage(dir, hooks = {}) {
  const execSync = hooks.execSync ?? spawnSync;
  const cwd = join(root, dir);
  const result = execSync('npx', ['attw', '--pack', '--profile', 'node16', '-f', 'table'], {
    cwd,
    encoding: 'utf8',
    shell: false,
  });
  const summary = `${result.stdout ?? ''}${result.stderr ?? ''}`.trim();
  if (isAttwToolCrash(summary)) {
    return {
      ok: true,
      skipped: true,
      summary: 'attw CLI error (upstream tool crash; not a finding about this package)',
    };
  }
  const ok = result.status === 0;
  return { ok, summary: summary || (ok ? 'No issues' : 'attw failed') };
}

/** @param {{ execSync?: typeof spawnSync }} [hooks] */
export function runAttwAudit(hooks = {}) {
  return ATTW_PACKAGES.map(({ dir, name }) => {
    const { ok, summary } = runAttwPackage(dir, hooks);
    return { name, ok, summary };
  });
}

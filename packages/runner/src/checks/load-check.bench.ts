import { resolve } from 'node:path';
import { fileURLToPath } from 'node:url';
import { bench, describe } from 'vitest';
import { loadCheck, resolveCheckNames } from './load-check.js';

const REPO_ROOT = resolve(fileURLToPath(new URL('../../../..', import.meta.url)));

const DEFAULT_CHECKS = [
  'read-repo-first',
  'changelog',
  'changelog-updated',
  'cspell',
  'eslint',
  'markdown-no-bold-italic',
  'prettier',
  'node-version',
  'markdown-front-matter',
  'semantic-commit',
  'vitest-coverage-exclude',
  'vitest-coverage-full',
] as const;

describe('loadCheck (publish/runtime)', () => {
  bench('changelog (light check)', async () => {
    await loadCheck('changelog', REPO_ROOT);
  });

  bench('node-version (light check)', async () => {
    await loadCheck('node-version', REPO_ROOT);
  });

  bench('eslint (heavy check)', async () => {
    await loadCheck('eslint', REPO_ROOT);
  });
});

describe('resolveCheckNames', () => {
  bench('default bundle list from repo root', async () => {
    await resolveCheckNames(REPO_ROOT);
  });
});

describe('default bundle sequential loadCheck', () => {
  bench('all 12 default checks', async () => {
    for (const name of DEFAULT_CHECKS) {
      await loadCheck(name, REPO_ROOT);
    }
  });
});

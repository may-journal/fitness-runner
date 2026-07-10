import { mkdtempSync, writeFileSync } from 'node:fs';
import { createRequire } from 'node:module';
import { tmpdir } from 'node:os';
import { join } from 'node:path';
import { PACKAGE_JSON, TSCONFIG_JSON } from './fileNames.js';
import { RUNNER_SKIP_DIRS } from './getSkipDirs.js';
import { SPEC_GLOB, TEST_GLOB } from './testGlobs.js';

const lintTsconfigCache = new Map<string, string>();

/** Temp tsconfig with absolute include/exclude so ESLint type-checks sources under root without a local tsconfig. */
export function resolveLintTsconfig(root: string, fitnessRunnerRoot: string): string {
  const cached = lintTsconfigCache.get(root);
  if (cached) return cached;

  const require = createRequire(join(fitnessRunnerRoot, PACKAGE_JSON));
  const base = require(join(fitnessRunnerRoot, 'tsconfig.lint.cjs')) as Record<string, unknown>;
  const exclude = [...RUNNER_SKIP_DIRS, TEST_GLOB, SPEC_GLOB].map((p) => join(root, p));
  const dir = mkdtempSync(join(tmpdir(), 'fitness-eslint-'));
  const path = join(dir, TSCONFIG_JSON);
  writeFileSync(
    path,
    JSON.stringify({
      ...base,
      exclude,
      include: [join(root, '**/*.ts')],
    })
  );
  lintTsconfigCache.set(root, path);
  return path;
}

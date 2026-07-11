import {
  DTS_GLOB,
  SPEC_GLOB,
  TEST_GLOB,
  TYPES_GLOB,
  checkResult,
  getCoverageExcludeFromConfig,
  getFitnessRunnerRoot,
  loadVitestConfig,
} from '@mayjournal/fitness-shared';
import type { VitestConfigRaw } from '@mayjournal/fitness-shared';
import { CheckName } from '@mayjournal/fitness';
import type { Check } from '@mayjournal/fitness';

export { configFromMod, VITEST_CONFIG_NAMES } from '@mayjournal/fitness-shared';

/** Allowed coverage exclude patterns: declaration, type-only, barrel index, worker entry; Vitest excludes tests by default. */
export const ALLOWED_COVERAGE_EXCLUDE_PATTERNS = [
  '**/*.bench.ts',
  DTS_GLOB,
  TYPES_GLOB,
  TEST_GLOB,
  SPEC_GLOB,
  '**/index.ts',
  '**/run-one-check-worker.ts',
] as const;

/** Allowed coverage exclude suffixes: declaration, type-only, test files, barrel index, worker entry. */
export const ALLOWED_SUFFIXES =
  /(\.(bench\.ts|d\.ts|types\.ts|test\.ts|spec\.ts)|index\.ts|run-one-check-worker\.ts)$/;

/** Returns true if the exclude pattern is not in the allowed conventional set (.d.ts, .types.ts, .test.ts, .spec.ts). */
function isDisallowedTsPattern(pattern: string): boolean {
  const t = pattern.trim();
  if (t.endsWith('/**')) return true;
  if (!t.includes('.ts')) return false;
  if (ALLOWED_SUFFIXES.test(t)) return false;
  return true;
}

/** Get coverage exclude array from Vitest config (test.coverage.exclude or coverage.exclude). */
export function getCoverageExclude(config: VitestConfigRaw | null): string[] {
  if (config == null) return [];
  const exclude = getCoverageExcludeFromConfig(config);
  if (!Array.isArray(exclude)) return [];
  return exclude;
}

/** Ensures Vitest coverage exclude only uses conventional patterns (e.g. *.d.ts, *.types.ts); Vitest excludes tests by default. */
export const vitestCoverageExcludeCheck: Check = {
  name: CheckName.VitestCoverageExclude,
  async run(root = process.cwd()) {
    const config = loadVitestConfig(root, getFitnessRunnerRoot());
    const exclude = getCoverageExclude(config);
    if (exclude.length === 0) return checkResult(true, [], 1);
    const bad = exclude.filter((p) => typeof p === 'string' && isDisallowedTsPattern(p));
    if (bad.length === 0) return checkResult(true, [], 1);
    const allowed = ALLOWED_COVERAGE_EXCLUDE_PATTERNS.join(', ');
    const errors = bad.map(
      (p) => `Vitest coverage exclude only allows ${allowed}; disallowed: ${JSON.stringify(p)}`
    );
    return checkResult(false, errors, 1);
  },
};

export default vitestCoverageExcludeCheck;

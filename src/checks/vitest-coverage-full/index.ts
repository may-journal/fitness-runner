import { existsSync, readFileSync } from 'node:fs';
import { execSync } from 'node:child_process';
import { createRequire } from 'node:module';
import { dirname, join } from 'node:path';
import { fileURLToPath } from 'node:url';
import { checkResult } from '../../utils/checkResult.js';
import { getExecSync } from '../../utils/runContext.js';
import type { ExecSyncFn } from '../../utils/runContext.js';
import { CheckName } from '../../types/index.types.js';
import type { Check } from '../../types/index.types.js';
import { enUS } from './enUS.js';
import { VITEST_CONFIG_NAMES } from '../vitest-coverage-exclude/index.js';

const require = createRequire(import.meta.url);
const VITEST_COVERAGE_CMD = 'npx vitest run --coverage';
const EXEC_OPTS = { encoding: 'utf8' as const, maxBuffer: 1024 * 1024 };
const REQUIRED_THRESHOLD = 100;

type VitestConfigWithThresholds = {
  coverage?: {
    thresholds?: { branches?: number; functions?: number; lines?: number; statements?: number };
  };
  test?: {
    coverage?: {
      thresholds?: { branches?: number; functions?: number; lines?: number; statements?: number };
    };
  };
};

/** True if x is a non-null object. */
function isObject(x: unknown): x is object {
  return typeof x === 'object' && x != null;
}

/** Extract config from loaded module (default export or module). */
function configFromMod(mod: unknown): VitestConfigWithThresholds | null {
  if (!isObject(mod)) return null;
  const def = (mod as { default?: unknown }).default;
  if (isObject(def)) return def as VitestConfigWithThresholds;
  return mod as VitestConfigWithThresholds;
}

/** Try to load a single vitest.config.* file. */
function tryLoadConfigFile(root: string, name: string): VitestConfigWithThresholds | null {
  const path = join(root, name);
  if (!existsSync(path)) return null;
  try {
    if (name.endsWith('.cjs') || name.endsWith('.js')) {
      return configFromMod(require(path));
    }
    const jiti = require('jiti')(root, { esmResolve: true });
    return configFromMod(jiti(path));
  } catch {
    return null;
  }
}

/** Try to load vitest config from package.json vitest key. */
function tryLoadPackageJsonVitest(root: string): VitestConfigWithThresholds | null {
  const pkgPath = join(root, 'package.json');
  if (!existsSync(pkgPath)) return null;
  try {
    const pkg = JSON.parse(readFileSync(pkgPath, 'utf8')) as {
      vitest?: VitestConfigWithThresholds;
    };
    return pkg.vitest != null && typeof pkg.vitest === 'object' ? pkg.vitest : null;
  } catch {
    return null;
  }
}

/** True if content has branches/functions/lines/statements all set to REQUIRED_THRESHOLD via regex. */
function contentHasAllThresholds(content: string): boolean {
  const has = (key: string) => new RegExp(`${key}:\\s*${REQUIRED_THRESHOLD}\\b`).test(content);
  return has('branches') && has('functions') && has('lines') && has('statements');
}

/** Fallback: read config file and detect thresholds at 100 via regex (avoids execute when loader fails). */
function tryParseThresholdsFromFile(root: string): VitestConfigWithThresholds | null {
  const fullThresholds = {
    test: {
      coverage: { thresholds: { branches: 100, functions: 100, lines: 100, statements: 100 } },
    },
  };
  for (const name of VITEST_CONFIG_NAMES) {
    const path = join(root, name);
    if (!existsSync(path)) continue;
    try {
      const content = readFileSync(path, 'utf8');
      if (contentHasAllThresholds(content)) return fullThresholds;
    } catch {
      // ignore
    }
  }
  return null;
}

/** Load Vitest config from root; tries fast paths (regex, package.json) before jiti. */
function loadVitestConfig(root: string): VitestConfigWithThresholds | null {
  const parsed = tryParseThresholdsFromFile(root);
  if (parsed != null) return parsed;
  const pkg = tryLoadPackageJsonVitest(root);
  if (pkg != null) return pkg;
  for (const name of VITEST_CONFIG_NAMES) {
    const config = tryLoadConfigFile(root, name);
    if (config != null) return config;
  }
  return null;
}

/** Returns test.coverage or coverage from config. */
function getCoverageBlock(config: VitestConfigWithThresholds | null) {
  if (!config) return undefined;
  return config.test?.coverage ?? config.coverage;
}

/** Get coverage thresholds from config. */
function getThresholds(config: VitestConfigWithThresholds | null) {
  return getCoverageBlock(config)?.thresholds;
}

/** Returns true if Vitest config has coverage thresholds all set to 100. */
export function hasFullCoverageThresholds(root: string): boolean {
  const t = getThresholds(loadVitestConfig(root));
  const vals = [t?.branches, t?.functions, t?.lines, t?.statements];
  return vals.every((v) => v === REQUIRED_THRESHOLD);
}

/** Resolves the fitness-runner package root (directory containing package.json for this package). */
export function getFitnessRunnerRoot(): string {
  let dir = dirname(fileURLToPath(import.meta.url));
  while (dir !== dirname(dir)) {
    if (existsSync(join(dir, 'package.json'))) return dir;
    dir = dirname(dir);
  }
  return dir;
}

/** Run vitest with coverage; returns exit code and combined stdout/stderr. */
export function runVitestCoverage(
  root: string,
  execSyncFn: ExecSyncFn = execSync
): { exitCode: number; output: string } {
  try {
    const out = execSyncFn(VITEST_COVERAGE_CMD, { ...EXEC_OPTS, cwd: root });
    return { exitCode: 0, output: out };
  } catch (e: unknown) {
    const err = e as { status?: number; stderr?: string; stdout?: string };
    const out = [err.stdout, err.stderr].filter(Boolean).join('\n');
    return { exitCode: typeof err.status === 'number' ? err.status : 1, output: out };
  }
}

export { enUS } from './enUS.js';

/** Returns a failed CheckResult if root or fitness-runner thresholds are not 100; else null. */
function thresholdCheckResult(
  root: string,
  context: Parameters<Check['run']>[1]
): ReturnType<typeof checkResult> | null {
  if (!hasFullCoverageThresholds(root)) return checkResult(false, [enUS.ThresholdsNot100], 1);
  const frRoot = context?._fitnessRunnerRootForTesting ?? getFitnessRunnerRoot();
  if (!hasFullCoverageThresholds(frRoot))
    return checkResult(false, [enUS.FitnessRunnerThresholdsNot100], 1);
  return null;
}

/** Build failure result from coverage run output. */
function coverageFailureResult(output: string): ReturnType<typeof checkResult> {
  const lastLine = output.trim().split('\n').pop()?.trim();
  const msg = lastLine || enUS.FallbackRunHint;
  return checkResult(false, [msg], 1);
}

/** Ensures both consumer and fitness-runner have 100% thresholds and vitest run --coverage passes in root. */
export const vitestCoverageFullCheck: Check = {
  name: CheckName.VitestCoverageFull,
  async run(root = process.cwd(), context) {
    const thresholdFail = thresholdCheckResult(root, context);
    if (thresholdFail) return thresholdFail;
    const { exitCode, output } = runVitestCoverage(root, getExecSync(context));
    return exitCode === 0 ? checkResult(true, [], 1) : coverageFailureResult(output);
  },
  runInProcess: true,
};

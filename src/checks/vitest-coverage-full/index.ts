import { existsSync, readFileSync } from 'node:fs';
import { execSync } from 'node:child_process';
import { dirname, join } from 'node:path';
import { fileURLToPath } from 'node:url';
import { checkResult } from '../../utils/checkResult.js';
import { execSyncResult } from '../../utils/execSyncResult.js';
import { getExecSync } from '../../utils/runContext.js';
import type { ExecSyncFn } from '../../utils/runContext.js';
import { CheckName } from '../../types/index.types.js';
import type { Check } from '../../types/index.types.js';
import { enUS } from './enUS.js';
import {
  loadVitestConfig,
  getThresholdsFromConfig,
  type VitestConfigRaw,
  VITEST_CONFIG_NAMES,
} from '../vitest-config/index.js';

const VITEST_COVERAGE_CMD = 'npx vitest run --coverage';
const REQUIRED_THRESHOLD = 100;

/** True if content has branches/functions/lines/statements all set to REQUIRED_THRESHOLD via regex. */
function contentHasAllThresholds(content: string): boolean {
  const has = (key: string) => new RegExp(`${key}:\\s*${REQUIRED_THRESHOLD}\\b`).test(content);
  return has('branches') && has('functions') && has('lines') && has('statements');
}

/** Fallback: read config file and detect thresholds at 100 via regex (avoids execute when loader fails). */
function tryParseThresholdsFromFile(root: string): VitestConfigRaw | null {
  const fullThresholds: VitestConfigRaw = {
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

/** Load Vitest config from root; tries regex fast path then shared loader. */
function loadVitestConfigWithThresholds(root: string): VitestConfigRaw | null {
  return tryParseThresholdsFromFile(root) ?? loadVitestConfig(root);
}

/** Returns true if Vitest config has coverage thresholds all set to 100. */
export function hasFullCoverageThresholds(root: string): boolean {
  const config = loadVitestConfigWithThresholds(root);
  const t = getThresholdsFromConfig(config);
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
  return execSyncResult(root, VITEST_COVERAGE_CMD, execSyncFn);
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

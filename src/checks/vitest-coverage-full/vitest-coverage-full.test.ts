import { existsSync, mkdtempSync, writeFileSync } from 'node:fs';
import { join } from 'node:path';
import { tmpdir } from 'node:os';
import { describe, it, expect } from 'vitest';
import {
  enUS,
  getFitnessRunnerRoot,
  hasFullCoverageThresholds,
  runVitestCoverage,
  vitestCoverageFullCheck,
} from './index.js';

describe('vitestCoverageFullCheck', () => {
  it('fails with ThresholdsNot100 when config thresholds are not all 100', async () => {
    const dir = mkdtempSync(join(tmpdir(), 'vitest-full-'));
    writeFileSync(
      join(dir, 'vitest.config.js'),
      'module.exports = { test: { coverage: { thresholds: { branches: 90, functions: 100, lines: 100, statements: 100 } } } };'
    );
    const result = await vitestCoverageFullCheck.run(dir);
    expect(result.ok).toBe(false);
    expect(result.errors[0]).toBe(enUS.ThresholdsNot100);
  });

  it('fails with FitnessRunnerThresholdsNot100 when fitness-runner root has thresholds not 100', async () => {
    const consumerDir = mkdtempSync(join(tmpdir(), 'vitest-consumer-'));
    writeFileSync(
      join(consumerDir, 'vitest.config.js'),
      'module.exports = { test: { coverage: { thresholds: { branches: 100, functions: 100, lines: 100, statements: 100 } } } };'
    );
    const fakeRunnerDir = mkdtempSync(join(tmpdir(), 'vitest-runner-'));
    writeFileSync(
      join(fakeRunnerDir, 'vitest.config.js'),
      'module.exports = { test: { coverage: { thresholds: { branches: 90, functions: 100, lines: 100, statements: 100 } } } };'
    );
    const result = await vitestCoverageFullCheck.run(consumerDir, {
      _fitnessRunnerRootForTesting: fakeRunnerDir,
      _execSync: () => '',
    });
    expect(result.ok).toBe(false);
    expect(result.errors[0]).toBe(enUS.FitnessRunnerThresholdsNot100);
  });

  it('passes when vitest run --coverage exits 0', async () => {
    const result = await vitestCoverageFullCheck.run(process.cwd(), {
      _execSync: () => '',
    });
    expect(result.ok).toBe(true);
    expect(result.errors).toHaveLength(0);
  });

  it('fails when vitest run --coverage exits non-zero', async () => {
    const result = await vitestCoverageFullCheck.run(process.cwd(), {
      _execSync: () => {
        throw Object.assign(new Error('coverage'), {
          status: 1,
          stdout: '',
          stderr: 'Coverage for branches (90%) does not meet threshold (100%).',
        });
      },
    });
    expect(result.ok).toBe(false);
    expect(result.errors).toHaveLength(1);
    expect(result.errors[0]).toContain('100%');
  });

  it('uses fallback message when output is empty', async () => {
    const result = await vitestCoverageFullCheck.run(process.cwd(), {
      _execSync: () => {
        throw Object.assign(new Error('coverage'), { status: 1 });
      },
    });
    expect(result.ok).toBe(false);
    expect(result.errors[0]).toBe(enUS.FallbackRunHint);
  });
});

describe('runVitestCoverage', () => {
  it('returns exitCode 0 and output when command succeeds', () => {
    const { exitCode, output } = runVitestCoverage(process.cwd(), () => 'ok');
    expect(exitCode).toBe(0);
    expect(output).toBe('ok');
  });

  it('returns exitCode and combined stdout/stderr when command throws', () => {
    const { exitCode, output } = runVitestCoverage(process.cwd(), () => {
      throw Object.assign(new Error('fail'), { status: 1, stdout: 'out', stderr: 'err' });
    });
    expect(exitCode).toBe(1);
    expect(output).toBe('out\nerr');
  });

  it('returns exitCode 1 when thrown error has no status', () => {
    const { exitCode } = runVitestCoverage(process.cwd(), () => {
      throw Object.assign(new Error('fail'), { stdout: 'o', stderr: 'e' });
    });
    expect(exitCode).toBe(1);
  });
});

describe('hasFullCoverageThresholds', () => {
  it('returns true when config has branches, functions, lines, statements all 100', () => {
    expect(hasFullCoverageThresholds(process.cwd())).toBe(true);
  });

  it('returns false when any threshold is not 100', () => {
    const dir = mkdtempSync(join(tmpdir(), 'vitest-thresh-'));
    writeFileSync(
      join(dir, 'vitest.config.js'),
      'module.exports = { test: { coverage: { thresholds: { branches: 100, functions: 100, lines: 100, statements: 99 } } } };'
    );
    expect(hasFullCoverageThresholds(dir)).toBe(false);
  });

  it('returns false when thresholds are missing', () => {
    const dir = mkdtempSync(join(tmpdir(), 'vitest-thresh-'));
    writeFileSync(join(dir, 'vitest.config.js'), 'module.exports = { test: { coverage: {} } };');
    expect(hasFullCoverageThresholds(dir)).toBe(false);
  });
});

describe('getFitnessRunnerRoot', () => {
  it('returns a directory that contains package.json', () => {
    const root = getFitnessRunnerRoot();
    expect(existsSync(join(root, 'package.json'))).toBe(true);
  });

  it('returned path has 100% coverage thresholds (fitness-runner must stay configured)', () => {
    expect(hasFullCoverageThresholds(getFitnessRunnerRoot())).toBe(true);
  });
});

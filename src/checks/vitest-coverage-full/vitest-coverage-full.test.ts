import { existsSync, mkdtempSync, mkdirSync, writeFileSync } from 'node:fs';
import { join } from 'node:path';
import { tmpdir } from 'node:os';
import { describe, it, expect, vi } from 'vitest';
import {
  enUS,
  getFitnessRunnerRoot,
  hasFullCoverageThresholds,
  runVitestCoverage,
  vitestCoverageFullCheck,
} from './index.js';

vi.mock('node:fs', async (importOriginal) => {
  const fs = await importOriginal<typeof import('node:fs')>();
  return { ...fs, existsSync: vi.fn(fs.existsSync) };
});

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
    expect(result.errors[0]).toContain('100');
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

  it('returns false when package.json is invalid (tryLoadPackageJsonVitest catch)', () => {
    const dir = mkdtempSync(join(tmpdir(), 'vitest-thresh-'));
    writeFileSync(join(dir, 'package.json'), '{ invalid }');
    expect(hasFullCoverageThresholds(dir)).toBe(false);
  });

  it('returns false when package.json vitest is null or non-object (line 66 null branch)', () => {
    const dir = mkdtempSync(join(tmpdir(), 'vitest-thresh-'));
    writeFileSync(join(dir, 'package.json'), '{"vitest":null}');
    expect(hasFullCoverageThresholds(dir)).toBe(false);
  });

  it('returns true via tryParseThresholdsFromFile when config file exists but load throws', () => {
    const dir = mkdtempSync(join(tmpdir(), 'vitest-thresh-'));
    writeFileSync(
      join(dir, 'vitest.config.js'),
      'syntax error; but branches: 100, functions: 100, lines: 100, statements: 100'
    );
    expect(hasFullCoverageThresholds(dir)).toBe(true);
  });

  it('returns true when package.json vitest has top-level coverage (no test key)', () => {
    const dir = mkdtempSync(join(tmpdir(), 'vitest-thresh-'));
    writeFileSync(
      join(dir, 'package.json'),
      '{"vitest":{"coverage":{"thresholds":{"branches":100,"functions":100,"lines":100,"statements":100}}}}'
    );
    expect(hasFullCoverageThresholds(dir)).toBe(true);
  });

  it('returns true via tryParseThresholdsFromFile when first config file is module.exports = null (configFromMod null path)', () => {
    const dir = mkdtempSync(join(tmpdir(), 'vitest-thresh-'));
    writeFileSync(
      join(dir, 'vitest.config.js'),
      'module.exports = null; // branches: 100, functions: 100, lines: 100, statements: 100'
    );
    expect(hasFullCoverageThresholds(dir)).toBe(true);
  });

  it('returns true when first config path exists but readFileSync throws (tryParseThresholdsFromFile catch)', () => {
    const dir = mkdtempSync(join(tmpdir(), 'vitest-thresh-'));
    mkdirSync(join(dir, 'vitest.config.cjs'));
    writeFileSync(
      join(dir, 'vitest.config.js'),
      'branches: 100, functions: 100, lines: 100, statements: 100'
    );
    expect(hasFullCoverageThresholds(dir)).toBe(true);
  });

  it('returns true when config has ESM default export (configFromMod returns def)', () => {
    const dir = mkdtempSync(join(tmpdir(), 'vitest-thresh-'));
    writeFileSync(
      join(dir, 'vitest.config.mjs'),
      'export default { test: { coverage: { thresholds: { branches: 100, functions: 100, lines: 100, statements: 100 } } } };'
    );
    expect(hasFullCoverageThresholds(dir)).toBe(true);
  });

  it('returns true when only vitest.config.mjs has thresholds via variable (tryLoadConfigFile jiti path)', () => {
    const dir = mkdtempSync(join(tmpdir(), 'vitest-thresh-'));
    writeFileSync(
      join(dir, 'vitest.config.mjs'),
      'const t = 100; export default { test: { coverage: { thresholds: { branches: t, functions: t, lines: t, statements: t } } } };'
    );
    expect(hasFullCoverageThresholds(dir)).toBe(true);
  });

  it('returns false when only vitest.config.js exists and throws on load (tryLoadConfigFile catch)', () => {
    const dir = mkdtempSync(join(tmpdir(), 'vitest-thresh-'));
    writeFileSync(join(dir, 'vitest.config.js'), 'module.exports = { x');
    expect(hasFullCoverageThresholds(dir)).toBe(false);
  });

  it('returns false when vitest.config.js exports non-object (configFromMod null branch)', () => {
    const dir = mkdtempSync(join(tmpdir(), 'vitest-thresh-'));
    writeFileSync(join(dir, 'vitest.config.js'), 'module.exports = null;');
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

  it('returns dir when no package.json in ancestor (exit loop)', async () => {
    const fs = await import('node:fs');
    vi.mocked(fs.existsSync).mockImplementation(() => false);
    try {
      const root = getFitnessRunnerRoot();
      expect(root).toBeDefined();
    } finally {
      vi.mocked(fs.existsSync).mockRestore();
    }
  });
});

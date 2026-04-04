import { mkdtempSync, mkdirSync, writeFileSync } from 'node:fs';
import { join } from 'node:path';
import { tmpdir } from 'node:os';
import { beforeEach, describe, it, expect, vi } from 'vitest';
import {
  eslintCheck,
  ESLINT_FALLBACK_MESSAGE,
  formatMessage,
  runEslintViaAPI,
  tryParseJsonArray,
} from './index';

describe('eslintCheck', () => {
  const mockRun = vi.fn<
    (
      root: string,
      paths: string[],
      frRoot?: string
    ) => Promise<{
      errors: string[];
      exitCode: number;
      filesChecked: number;
    }>
  >();

  beforeEach(() => {
    mockRun.mockReset();
  });

  it('passes when ESLint reports no issues', async () => {
    mockRun.mockResolvedValue({ errors: [], exitCode: 0, filesChecked: 0 });
    const dir = mkdtempSync(join(tmpdir(), 'eslint-'));
    const result = await eslintCheck.run(dir, { _eslintRunForTesting: mockRun });
    expect(result.ok).toBe(true);
    expect(result.errors).toHaveLength(0);
    expect(result.meta?.filesChecked).toBe(0);
    expect(mockRun).toHaveBeenCalledWith(dir, ['.'], undefined);
  });

  it('fails and returns error lines when ESLint reports issues', async () => {
    mockRun.mockResolvedValue({
      errors: ['/repo/src/foo.ts:1:1 - Unexpected var. (no-var)'],
      exitCode: 1,
      filesChecked: 1,
    });
    const dir = mkdtempSync(join(tmpdir(), 'eslint-'));
    const result = await eslintCheck.run(dir, { _eslintRunForTesting: mockRun });
    expect(result.ok).toBe(false);
    expect(result.errors.some((e) => e.includes('no-var') && e.includes('Unexpected var'))).toBe(
      true
    );
    expect(result.meta?.filesChecked).toBe(1);
  });

  it('uses staged paths when context has stagedFiles', async () => {
    mockRun.mockResolvedValue({ errors: [], exitCode: 0, filesChecked: 0 });
    const dir = mkdtempSync(join(tmpdir(), 'eslint-'));
    writeFileSync(join(dir, 'bar.ts'), 'x');
    await eslintCheck.run(dir, { stagedFiles: ['bar.ts'], _eslintRunForTesting: mockRun });
    expect(mockRun).toHaveBeenCalledWith(dir, ['bar.ts'], undefined);
  });

  it('uses fitness runner root from context for testing', async () => {
    mockRun.mockResolvedValue({ errors: [], exitCode: 0, filesChecked: 0 });
    const dir = mkdtempSync(join(tmpdir(), 'eslint-'));
    const fakeRoot = mkdtempSync(join(tmpdir(), 'fitness-root-'));
    await eslintCheck.run(dir, {
      _fitnessRunnerRootForTesting: fakeRoot,
      _eslintRunForTesting: mockRun,
    });
    expect(mockRun).toHaveBeenCalledWith(dir, ['.'], fakeRoot);
  });

  it('returns fallback when run throws', async () => {
    mockRun.mockRejectedValue(new Error('Config not found'));
    const dir = mkdtempSync(join(tmpdir(), 'eslint-'));
    const result = await eslintCheck.run(dir, { _eslintRunForTesting: mockRun });
    expect(result.ok).toBe(false);
    expect(result.errors[0]).toBe(ESLINT_FALLBACK_MESSAGE);
    expect(result.errors[1]).toContain('Config not found');
  });

  it('returns fallback when run throws a non-Error value', async () => {
    mockRun.mockRejectedValue('string failure');
    const dir = mkdtempSync(join(tmpdir(), 'eslint-'));
    const result = await eslintCheck.run(dir, { _eslintRunForTesting: mockRun });
    expect(result.ok).toBe(false);
    expect(result.errors[1]).toContain('string failure');
  });

  it('formatMessage includes ruleId when present', () => {
    expect(formatMessage('/f.ts', { line: 1, column: 2, message: 'x', ruleId: 'no-var' })).toBe(
      '/f.ts:1:2 - x (no-var)'
    );
  });

  it('formatMessage uses 0 for missing line/column and empty string for missing ruleId', () => {
    expect(formatMessage('/f.ts', { message: 'y', ruleId: null })).toBe('/f.ts:0:0 - y');
  });

  it('tryParseJsonArray returns null for valid JSON that is not an array', () => {
    expect(tryParseJsonArray('{}')).toBeNull();
  });

  it('tryParseJsonArray returns null when input is not valid JSON', () => {
    expect(tryParseJsonArray('not json')).toBeNull();
  });

  it('tryParseJsonArray returns array for valid JSON array', () => {
    const data = tryParseJsonArray(
      '[{"errorCount":0,"filePath":"/a.ts","messages":[],"warningCount":0}]'
    );
    expect(Array.isArray(data)).toBe(true);
    expect(data).toHaveLength(1);
  });

  it('formats message without ruleId when ruleId is null', async () => {
    mockRun.mockResolvedValue({
      errors: ['/repo/a.ts:2:3 - Some error'],
      exitCode: 1,
      filesChecked: 1,
    });
    const dir = mkdtempSync(join(tmpdir(), 'eslint-'));
    const result = await eslintCheck.run(dir, { _eslintRunForTesting: mockRun });
    expect(result.ok).toBe(false);
    expect(result.errors[0]).toBe('/repo/a.ts:2:3 - Some error');
  });

  it('runEslintViaAPI with empty paths lints "."', async () => {
    const dir = mkdtempSync(join(tmpdir(), 'eslint-'));
    const result = await runEslintViaAPI(dir, []);
    expect(result.exitCode).toBe(0);
    expect(result.filesChecked).toBeGreaterThanOrEqual(0);
  });

  it('check uses runEslintViaAPI when _eslintRunForTesting is absent', async () => {
    const dir = mkdtempSync(join(tmpdir(), 'eslint-'));
    const result = await eslintCheck.run(dir);
    expect(result.ok).toBe(true);
  });

  it('runEslintViaAPI maps ESLint messages through formatMessage', async () => {
    const dir = mkdtempSync(join(tmpdir(), 'eslint-'));
    writeFileSync(join(dir, 'bad.js'), 'const o = { z: 1, a: 2 };\n');
    const result = await runEslintViaAPI(dir, ['bad.js']);
    expect(result.exitCode).toBe(1);
    expect(result.errors.some((e) => e.includes('bad.js') && e.includes('sort-keys'))).toBe(true);
  });

  it('passes staged path with quotes to runner', async () => {
    mockRun.mockResolvedValue({ errors: [], exitCode: 0, filesChecked: 0 });
    const dir = mkdtempSync(join(tmpdir(), 'eslint-'));
    const pathWithQuote = 'src/bar "quoted".ts';
    mkdirSync(join(dir, 'src'), { recursive: true });
    writeFileSync(join(dir, pathWithQuote), 'x');
    await eslintCheck.run(dir, {
      stagedFiles: [pathWithQuote],
      _eslintRunForTesting: mockRun,
    });
    expect(mockRun).toHaveBeenCalledWith(dir, [pathWithQuote], undefined);
  });
});

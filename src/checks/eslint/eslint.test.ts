import { mkdtempSync, mkdirSync, writeFileSync } from 'node:fs';
import { join } from 'node:path';
import { tmpdir } from 'node:os';
import { execSync } from 'node:child_process';
import { beforeEach, describe, it, expect, vi } from 'vitest';
import {
  eslintCheck,
  ESLINT_CLI,
  ESLINT_FALLBACK_MESSAGE,
  formatMessage,
  runEslint,
  tryParseJsonArray,
} from './index';

vi.mock('node:child_process', async (importOriginal) => {
  const mod = await importOriginal<typeof import('node:child_process')>();
  return { ...mod, execSync: vi.fn(mod.execSync) };
});

describe('eslintCheck', () => {
  beforeEach(() => {
    vi.mocked(execSync).mockReset();
  });

  it('passes when ESLint reports no issues', async () => {
    vi.mocked(execSync).mockReturnValue('[]');
    const dir = mkdtempSync(join(tmpdir(), 'eslint-'));
    const result = await eslintCheck.run(dir);
    expect(result.ok).toBe(true);
    expect(result.errors).toHaveLength(0);
    expect(result.meta?.filesChecked).toBe(0);
  });

  it('fails and returns error lines when ESLint reports issues', async () => {
    const json = JSON.stringify([
      {
        filePath: '/repo/src/foo.ts',
        messages: [
          { line: 1, column: 1, message: 'Unexpected var.', ruleId: 'no-var', severity: 2 },
        ],
        errorCount: 1,
        warningCount: 0,
      },
    ]);
    vi.mocked(execSync).mockImplementation(() => {
      throw Object.assign(new Error(), { stdout: json, status: 1 });
    });
    const dir = mkdtempSync(join(tmpdir(), 'eslint-'));
    const result = await eslintCheck.run(dir);
    expect(result.ok).toBe(false);
    expect(result.errors.some((e) => e.includes('no-var') && e.includes('Unexpected var'))).toBe(
      true
    );
    expect(result.meta?.filesChecked).toBe(1);
  });

  it('uses staged paths when context has stagedFiles', async () => {
    vi.mocked(execSync).mockReturnValue('[]');
    const dir = mkdtempSync(join(tmpdir(), 'eslint-'));
    writeFileSync(join(dir, 'bar.ts'), 'x');
    const result = await eslintCheck.run(dir, { stagedFiles: ['bar.ts'] });
    expect(result.ok).toBe(true);
    const calls = vi.mocked(execSync).mock.calls;
    const lastCall = calls[calls.length - 1][0];
    expect(lastCall).toContain(ESLINT_CLI);
    expect(lastCall).toContain('bar.ts');
  });

  it('parses JSON array when output has leading stderr (extracts file/line errors)', async () => {
    const json = JSON.stringify([
      {
        filePath: '/repo/src/foo.ts',
        messages: [
          { line: 10, column: 5, message: 'Unexpected var.', ruleId: 'no-var', severity: 2 },
        ],
        errorCount: 1,
        warningCount: 0,
      },
    ]);
    vi.mocked(execSync).mockImplementation(() => {
      throw Object.assign(new Error(), {
        stdout: `Warning: Some config message\n${json}`,
        status: 1,
      });
    });
    const dir = mkdtempSync(join(tmpdir(), 'eslint-'));
    const result = await eslintCheck.run(dir);
    expect(result.ok).toBe(false);
    expect(result.errors[0]).toBe('/repo/src/foo.ts:10:5 - Unexpected var. (no-var)');
    expect(result.meta?.filesChecked).toBe(1);
  });

  it('returns fallback when output has ] before [ (no valid array span)', async () => {
    vi.mocked(execSync).mockImplementation(() => {
      throw Object.assign(new Error(), { stdout: 'a]b[c', status: 1 });
    });
    const dir = mkdtempSync(join(tmpdir(), 'eslint-'));
    const result = await eslintCheck.run(dir);
    expect(result.ok).toBe(false);
    expect(result.errors[0]).toBe(ESLINT_FALLBACK_MESSAGE);
  });

  it('returns fallback when output has brackets but invalid JSON between them', async () => {
    vi.mocked(execSync).mockImplementation(() => {
      throw Object.assign(new Error(), { stdout: 'x[} ]y', status: 1 });
    });
    const dir = mkdtempSync(join(tmpdir(), 'eslint-'));
    const result = await eslintCheck.run(dir);
    expect(result.ok).toBe(false);
    expect(result.errors[0]).toBe(ESLINT_FALLBACK_MESSAGE);
  });

  it('returns fallback error when exit non-zero and output is not valid JSON', async () => {
    vi.mocked(execSync).mockImplementation(() => {
      throw Object.assign(new Error(), { stdout: 'No ESLint config found', status: 2 });
    });
    const dir = mkdtempSync(join(tmpdir(), 'eslint-'));
    const result = await eslintCheck.run(dir);
    expect(result.ok).toBe(false);
    expect(result.errors[0]).toBe(ESLINT_FALLBACK_MESSAGE);
  });

  it('runEslint catch uses status 1 when err.status is not a number', async () => {
    vi.mocked(execSync).mockImplementation(() => {
      throw Object.assign(new Error(), { stdout: 'x', stderr: 'y' });
    });
    const dir = mkdtempSync(join(tmpdir(), 'eslint-'));
    const result = await eslintCheck.run(dir);
    expect(result.ok).toBe(false);
    expect(result.errors[0]).toBe(ESLINT_FALLBACK_MESSAGE);
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

  it('formats message without ruleId when ruleId is null', async () => {
    const json = JSON.stringify([
      {
        filePath: '/repo/a.ts',
        messages: [{ line: 2, column: 3, message: 'Some error', ruleId: null, severity: 2 }],
        errorCount: 1,
        warningCount: 0,
      },
    ]);
    vi.mocked(execSync).mockImplementation(() => {
      throw Object.assign(new Error(), { stdout: json, status: 1 });
    });
    const dir = mkdtempSync(join(tmpdir(), 'eslint-'));
    const result = await eslintCheck.run(dir);
    expect(result.ok).toBe(false);
    expect(result.errors[0]).toBe('/repo/a.ts:2:3 - Some error');
  });

  it('parseJsonResults returns empty when output is valid JSON but not array', async () => {
    vi.mocked(execSync).mockReturnValue('{}');
    const dir = mkdtempSync(join(tmpdir(), 'eslint-'));
    const result = await eslintCheck.run(dir);
    expect(result.ok).toBe(true);
    expect(result.errors).toHaveLength(0);
    expect(result.meta?.filesChecked).toBe(0);
  });

  it('runEslint with empty paths uses "." as args', () => {
    vi.mocked(execSync).mockReturnValue('[]');
    const dir = mkdtempSync(join(tmpdir(), 'eslint-'));
    runEslint(dir, [], execSync);
    const calls = vi.mocked(execSync).mock.calls;
    const call = calls[calls.length - 1][0];
    expect(call).toContain(ESLINT_CLI);
    expect(call).toMatch(/\beslint\s+\.\s+--format/);
  });

  it('escapes quotes in path when building eslint args', async () => {
    vi.mocked(execSync).mockReturnValue('[]');
    const dir = mkdtempSync(join(tmpdir(), 'eslint-'));
    const pathWithQuote = 'src/bar "quoted".ts';
    mkdirSync(join(dir, 'src'), { recursive: true });
    writeFileSync(join(dir, pathWithQuote), 'x');
    await eslintCheck.run(dir, { stagedFiles: [pathWithQuote] });
    const calls = vi.mocked(execSync).mock.calls;
    const lastCall = calls[calls.length - 1][0];
    expect(lastCall).toContain(ESLINT_CLI);
    expect(lastCall).toContain('\\"');
  });
});

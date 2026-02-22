import { mkdtempSync, writeFileSync } from 'node:fs';
import { join } from 'node:path';
import { tmpdir } from 'node:os';
import { execSync } from 'node:child_process';
import { beforeEach, describe, it, expect, vi } from 'vitest';
import {
  prettierCheck,
  PRETTIER_CLI,
  PRETTIER_FALLBACK_MESSAGE,
  hasPrettierConfig,
  runPrettier,
  parsePrettierOutput,
} from './index';

vi.mock('node:child_process', async (importOriginal) => {
  const mod = await importOriginal<typeof import('node:child_process')>();
  return { ...mod, execSync: vi.fn(mod.execSync) };
});

describe('prettierCheck', () => {
  beforeEach(() => {
    vi.mocked(execSync).mockReset();
  });

  it('skips when no Prettier config', async () => {
    const dir = mkdtempSync(join(tmpdir(), 'prettier-'));
    const result = await prettierCheck.run(dir);
    expect(result.ok).toBe(true);
    expect(result.errors).toHaveLength(0);
    expect(result.meta?.filesChecked).toBe(0);
    expect(execSync).not.toHaveBeenCalled();
  });

  it('passes when Prettier reports no issues', async () => {
    const dir = mkdtempSync(join(tmpdir(), 'prettier-'));
    writeFileSync(join(dir, '.prettierrc.json'), '{}');
    vi.mocked(execSync).mockReturnValue(
      'Checking formatting...\nAll matched files use Prettier code style!'
    );
    const result = await prettierCheck.run(dir);
    expect(result.ok).toBe(true);
    expect(result.errors).toHaveLength(0);
  });

  it('fails and returns file paths when Prettier reports issues', async () => {
    const dir = mkdtempSync(join(tmpdir(), 'prettier-'));
    writeFileSync(join(dir, '.prettierrc.json'), '{}');
    const output =
      'Checking formatting...\n[warn] src/foo.ts\n[warn] Code style issues found in 1 file above. Run Prettier with --write to fix.';
    vi.mocked(execSync).mockImplementation(() => {
      throw Object.assign(new Error(), { stdout: output, status: 1 });
    });
    const result = await prettierCheck.run(dir);
    expect(result.ok).toBe(false);
    expect(result.errors).toContain('src/foo.ts');
    expect(result.meta?.filesChecked).toBe(1);
  });

  it('uses staged paths when context has stagedFiles', async () => {
    const dir = mkdtempSync(join(tmpdir(), 'prettier-'));
    writeFileSync(join(dir, '.prettierrc.json'), '{}');
    writeFileSync(join(dir, 'bar.ts'), 'x');
    vi.mocked(execSync).mockReturnValue('All matched files use Prettier code style!');
    const result = await prettierCheck.run(dir, { stagedFiles: ['bar.ts'] });
    expect(result.ok).toBe(true);
    const calls = vi.mocked(execSync).mock.calls;
    expect(calls[calls.length - 1][0]).toContain(PRETTIER_CLI);
    expect(calls[calls.length - 1][0]).toContain('bar.ts');
  });

  it('returns fallback error when exit non-zero and output has no [warn] paths', async () => {
    const dir = mkdtempSync(join(tmpdir(), 'prettier-'));
    writeFileSync(join(dir, '.prettierrc.json'), '{}');
    vi.mocked(execSync).mockImplementation(() => {
      throw Object.assign(new Error(), { stdout: 'Prettier failed', status: 2 });
    });
    const result = await prettierCheck.run(dir);
    expect(result.ok).toBe(false);
    expect(result.errors[0]).toBe(PRETTIER_FALLBACK_MESSAGE);
  });

  it('parsePrettierOutput extracts file paths from [warn] lines', () => {
    const output = '[warn] a.ts\n[warn] b.ts\n[warn] Code style issues found in 2 files above.';
    expect(parsePrettierOutput(output)).toEqual(['a.ts', 'b.ts']);
  });

  it('hasPrettierConfig returns true when .prettierrc.json exists', () => {
    const dir = mkdtempSync(join(tmpdir(), 'prettier-'));
    writeFileSync(join(dir, '.prettierrc.json'), '{}');
    expect(hasPrettierConfig(dir)).toBe(true);
  });

  it('hasPrettierConfig returns true when package.json has "prettier" field', async () => {
    const dir = mkdtempSync(join(tmpdir(), 'prettier-'));
    writeFileSync(join(dir, 'package.json'), '{"prettier":"@fitness/runner/prettier.config"}');
    expect(hasPrettierConfig(dir)).toBe(true);
    vi.mocked(execSync).mockReturnValue('All matched files use Prettier code style!');
    const result = await prettierCheck.run(dir);
    expect(result.ok).toBe(true);
    expect(execSync).toHaveBeenCalled();
  });

  it('hasPrettierConfig returns false when package.json is invalid JSON', () => {
    const dir = mkdtempSync(join(tmpdir(), 'prettier-'));
    writeFileSync(join(dir, 'package.json'), 'not valid json');
    expect(hasPrettierConfig(dir)).toBe(false);
  });

  it('runPrettier with paths passes them to CLI', () => {
    const dir = mkdtempSync(join(tmpdir(), 'prettier-'));
    vi.mocked(execSync).mockReturnValue('');
    runPrettier(dir, ['a.ts', 'b.json'], execSync);
    const call = vi.mocked(execSync).mock.calls[0][0];
    expect(call).toContain('--check');
    expect(call).toContain('a.ts');
    expect(call).toContain('b.json');
  });

  it('runPrettier with empty paths uses "."', () => {
    const dir = mkdtempSync(join(tmpdir(), 'prettier-'));
    vi.mocked(execSync).mockReturnValue('');
    runPrettier(dir, [], execSync);
    expect(vi.mocked(execSync).mock.calls[0][0]).toMatch(/prettier\s+--check\s+\./);
  });

  it('runPrettier with passthroughArgs forwards them to CLI without --check', () => {
    const dir = mkdtempSync(join(tmpdir(), 'prettier-'));
    vi.mocked(execSync).mockReturnValue('');
    runPrettier(dir, [], execSync, ['--write', '.']);
    const call = vi.mocked(execSync).mock.calls[0][0];
    expect(call).toContain('--write');
    expect(call).not.toContain('--check');
  });

  it('runPrettier uses exitCode 1 when err.status is not a number', async () => {
    const dir = mkdtempSync(join(tmpdir(), 'prettier-'));
    writeFileSync(join(dir, '.prettierrc.json'), '{}');
    vi.mocked(execSync).mockImplementation(() => {
      throw Object.assign(new Error(), { stdout: '' });
    });
    const result = await prettierCheck.run(dir);
    expect(result.ok).toBe(false);
    expect(result.errors[0]).toBe(PRETTIER_FALLBACK_MESSAGE);
  });
});

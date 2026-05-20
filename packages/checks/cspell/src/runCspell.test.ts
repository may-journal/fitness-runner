import { mkdtempSync, writeFileSync } from 'node:fs';
import { join } from 'node:path';
import { tmpdir } from 'node:os';
import { describe, it, expect, vi } from 'vitest';

vi.mock('cspell-lib', async (importOriginal) => {
  const actual = await importOriginal<typeof import('cspell-lib')>();
  const g = globalThis as { __cspellThrow?: boolean };
  return {
    ...actual,
    readConfigFile: vi.fn(() => Promise.resolve({})),
    spellCheckFile: vi.fn((..._args: unknown[]) => {
      if (g.__cspellThrow) {
        g.__cspellThrow = false;
        return Promise.reject(new Error());
      }
      return Promise.resolve({
        document: { text: 'x' },
        issues: [{ line: {}, message: undefined }],
      });
    }),
  };
});

import { spellCheckFile } from 'cspell-lib';
import { enUS, cspellCheck, runCspell } from './index.js';

describe('cspell top-level exports', () => {
  it('default export matches cspellCheck', async () => {
    const mod = await import('./index.js');
    expect(mod.default).toBe(mod.cspellCheck);
  });

  describe('enUS', () => {
    it('exposes FallbackRunHint', () => {
      expect(enUS.FallbackRunHint).toBe('cspell reported issues (run: npx cspell <files>)');
    });
  });

  describe('cspellCheck', () => {
    it('has name Cspell', () => {
      expect(cspellCheck.name).toBe('cspell');
    });

    it('run() passes with no cspell.json', async () => {
      const dir = mkdtempSync(join(tmpdir(), 'cspell-'));
      const result = await cspellCheck.run(dir);
      expect(result.ok).toBe(true);
      expect(result.errors).toHaveLength(0);
      expect(result.meta?.filesChecked).toBe(0);
    });

    it('run() CLI path uses buildCspellResult and parseFilesChecked', async () => {
      const dir = mkdtempSync(join(tmpdir(), 'cspell-'));
      writeFileSync(join(dir, 'cspell.json'), '{}');
      const result = await cspellCheck.run(dir, { _execSync: () => 'Files checked: 2\n' });
      expect(result.ok).toBe(true);
      expect(result.meta?.filesChecked).toBe(2);
    });

    it('run() CLI path parses issue lines into errors', async () => {
      const dir = mkdtempSync(join(tmpdir(), 'cspell-'));
      writeFileSync(join(dir, 'cspell.json'), '{}');
      writeFileSync(join(dir, 'f.md'), 'x');
      const result = await cspellCheck.run(dir, {
        _execSync: () => `${join(dir, 'f.md')}:1:1 - Unknown word: typo\n`,
      });
      expect(result.ok).toBe(false);
      expect(result.errors.some((e) => e.includes('typo'))).toBe(true);
    });

    it('run() CLI path returns FallbackRunHint when exec throws and no parseable issues', async () => {
      const dir = mkdtempSync(join(tmpdir(), 'cspell-'));
      writeFileSync(join(dir, 'cspell.json'), '{}');
      const err = new Error() as Error & { status?: number };
      err.status = 1;
      const result = await cspellCheck.run(dir, {
        _execSync: () => {
          throw err;
        },
      });
      expect(result.ok).toBe(false);
      expect(result.errors[0]).toBe(enUS.FallbackRunHint);
    });

    it('run() CLI path with stagedFiles runs on staged paths', async () => {
      const dir = mkdtempSync(join(tmpdir(), 'cspell-'));
      writeFileSync(join(dir, 'cspell.json'), '{}');
      writeFileSync(join(dir, 's.md'), 'x');
      const result = await cspellCheck.run(dir, {
        stagedFiles: ['s.md'],
        _execSync: (cmd) => (cmd.includes('s.md') ? `${join(dir, 's.md')}:1:1 - Unknown: x` : ''),
      });
      expect(result.ok).toBe(false);
    });

    it('run() CLI path with stagedFiles but none exist returns ok', async () => {
      const dir = mkdtempSync(join(tmpdir(), 'cspell-'));
      writeFileSync(join(dir, 'cspell.json'), '{}');
      const result = await cspellCheck.run(dir, {
        stagedFiles: ['missing.md'],
        _execSync: () => '',
      });
      expect(result.ok).toBe(true);
      expect(result.meta?.filesChecked).toBe(0);
    });

    it('run() CLI path skips staged .gitignore (ignorePaths basename)', async () => {
      const dir = mkdtempSync(join(tmpdir(), 'cspell-'));
      writeFileSync(join(dir, 'cspell.json'), '{}');
      writeFileSync(join(dir, '.gitignore'), '# Nuxt\n');
      const execSync = vi.fn(() => '');
      const result = await cspellCheck.run(dir, {
        stagedFiles: ['.gitignore', 'missing.md'],
        _execSync: execSync,
      });
      expect(result.ok).toBe(true);
      expect(execSync).not.toHaveBeenCalled();
    });

    it('run() lib path when spellCheckFile throws returns FallbackRunHint', async () => {
      (globalThis as { __cspellThrow?: boolean }).__cspellThrow = true;
      const dir = mkdtempSync(join(tmpdir(), 'cspell-'));
      writeFileSync(join(dir, 'cspell.json'), '{}');
      writeFileSync(join(dir, 'a.md'), 'x');
      const result = await cspellCheck.run(dir);
      expect(result.ok).toBe(false);
      expect(result.errors[0]).toContain(enUS.FallbackRunHint);
    });

    it('run() lib path with staged but none exist returns ok', async () => {
      const dir = mkdtempSync(join(tmpdir(), 'cspell-'));
      writeFileSync(join(dir, 'cspell.json'), '{}');
      const result = await cspellCheck.run(dir, { stagedFiles: ['nope.md'] });
      expect(result.ok).toBe(true);
      expect(result.meta?.filesChecked).toBe(0);
    });

    it('run() lib path returns mapped issues from multiple files', async () => {
      const dir = mkdtempSync(join(tmpdir(), 'cspell-'));
      writeFileSync(join(dir, 'cspell.json'), '{}');
      writeFileSync(join(dir, 'one.md'), 'a');
      writeFileSync(join(dir, 'two.md'), 'b');
      const result = await cspellCheck.run(dir);
      expect(result.ok).toBe(false);
      expect(result.errors).toHaveLength(2);
      expect(result.meta?.filesChecked).toBe(2);
    });

    it('run() lib path formats issue with offset in line (offsetToLineCol col branch)', async () => {
      const dir = mkdtempSync(join(tmpdir(), 'cspell-'));
      writeFileSync(join(dir, 'cspell.json'), '{}');
      writeFileSync(join(dir, 'f.md'), 'xy');
      vi.mocked(spellCheckFile).mockResolvedValueOnce({
        document: { text: 'xy' },
        issues: [{ line: { offset: 1 }, message: 'Unknown word', text: 'x' }],
      });
      const result = await cspellCheck.run(dir);
      expect(result.ok).toBe(false);
      expect(result.errors[0]).toMatch(/f\.md:1:2.*Unknown word/);
    });

    it('run() lib path formats issue after newline (offsetToLineCol newline branch)', async () => {
      const dir = mkdtempSync(join(tmpdir(), 'cspell-'));
      writeFileSync(join(dir, 'cspell.json'), '{}');
      writeFileSync(join(dir, 'g.md'), 'a\nb');
      vi.mocked(spellCheckFile).mockResolvedValueOnce({
        document: { text: 'a\nb' },
        issues: [{ line: { offset: 2 }, message: 'Unknown word', text: 'b' }],
      });
      const result = await cspellCheck.run(dir);
      expect(result.ok).toBe(false);
      expect(result.errors[0]).toMatch(/g\.md:2:1.*Unknown word/);
    });

    it('run() lib path handles result.document without text', async () => {
      const dir = mkdtempSync(join(tmpdir(), 'cspell-'));
      writeFileSync(join(dir, 'cspell.json'), '{}');
      writeFileSync(join(dir, 'h.md'), 'x');
      vi.mocked(spellCheckFile).mockResolvedValueOnce({
        document: {},
        issues: [{ line: { offset: 0 }, message: 'bad' }],
      });
      const result = await cspellCheck.run(dir);
      expect(result.ok).toBe(false);
      expect(result.errors[0]).toMatch(/h\.md:1:1.*bad/);
    });
  });

  describe('runCspell', () => {
    it('returns exitCode 0 and empty output when paths is empty', () => {
      const result = runCspell('/tmp', []);
      expect(result.exitCode).toBe(0);
      expect(result.output).toBe('');
    });

    it('returns exec output when mock exec succeeds', () => {
      const dir = mkdtempSync(join(tmpdir(), 'cspell-'));
      writeFileSync(join(dir, 'a.md'), 'x');
      const result = runCspell(dir, [join(dir, 'a.md')], () => 'Files checked: 1\n');
      expect(result.exitCode).toBe(0);
      expect(result.output).toBe('Files checked: 1\n');
    });

    it('returns exitCode and stderr when exec throws', () => {
      const dir = mkdtempSync(join(tmpdir(), 'cspell-'));
      writeFileSync(join(dir, 'a.md'), 'x');
      const err = new Error() as Error & { status?: number; stderr?: string };
      err.status = 2;
      err.stderr = 'stderr only';
      const result = runCspell(dir, [join(dir, 'a.md')], () => {
        throw err;
      });
      expect(result.exitCode).toBe(2);
      expect(result.output).toBe('stderr only');
    });

    it('returns exitCode 1 and stdout+stderr when exec throws with no status', () => {
      const dir = mkdtempSync(join(tmpdir(), 'cspell-'));
      writeFileSync(join(dir, 'a.md'), 'x');
      const err = new Error() as Error & { stdout?: string; stderr?: string };
      err.stdout = 'out';
      err.stderr = 'err';
      const result = runCspell(dir, [join(dir, 'a.md')], () => {
        throw err;
      });
      expect(result.exitCode).toBe(1);
      expect(result.output).toBe('out\nerr');
    });
  });
});

import { mkdtempSync, mkdirSync, writeFileSync } from 'node:fs';
import { join } from 'node:path';
import { tmpdir } from 'node:os';
import { describe, it, expect, vi } from 'vitest';
vi.mock('cspell-lib', async (importOriginal) => {
  const actual = await importOriginal<typeof import('cspell-lib')>();
  return {
    ...actual,
    spellCheckFile: vi.fn((...args: unknown[]) => {
      const g = globalThis as { __cspellThrowOnce?: boolean; __cspellReturnNoTextIssue?: boolean };
      if (g.__cspellThrowOnce) {
        g.__cspellThrowOnce = false;
        return Promise.reject(new Error());
      }
      if (g.__cspellReturnNoTextIssue) {
        g.__cspellReturnNoTextIssue = false;
        return Promise.resolve({
          issues: [{ message: 'Unknown', line: {} }],
          document: {},
          settingsUsed: {},
          localConfigFilepath: undefined,
          options: {},
          checked: true,
        });
      }
      return actual.spellCheckFile(args[0] as string, args[1] as object, args[2] as object);
    }),
  };
});

import { cspellCheck, runCspell } from './index.js';

describe('cspellCheck', () => {
  it('passes when cspell.json is missing', async () => {
    const dir = mkdtempSync(join(tmpdir(), 'cspell-'));
    const result = await cspellCheck.run(dir);
    expect(result.ok).toBe(true);
    expect(result.meta?.filesChecked).toBe(0);
  });

  it('fails when file has unknown word', async () => {
    const dir = mkdtempSync(join(tmpdir(), 'cspell-'));
    writeFileSync(join(dir, 'cspell.json'), '{"version":"0.2","words":[]}');
    writeFileSync(join(dir, 'package.json'), '{"devDependencies":{"cspell":"^8.0.0"}}');
    writeFileSync(join(dir, 'doc.md'), 'teh quick broown');
    const result = await cspellCheck.run(dir);
    expect(result.ok).toBe(false);
    expect(result.errors.length).toBeGreaterThan(0);
    expect(result.errors.some((e) => e.includes('teh') || e.includes('broown'))).toBe(true);
  });

  it('passes when file has no unknown words', async () => {
    const dir = mkdtempSync(join(tmpdir(), 'cspell-'));
    writeFileSync(join(dir, 'cspell.json'), '{"version":"0.2","words":[]}');
    writeFileSync(join(dir, 'package.json'), '{"devDependencies":{"cspell":"^8.0.0"}}');
    writeFileSync(join(dir, 'doc.md'), 'the quick brown');
    const result = await cspellCheck.run(dir);
    expect(result.ok).toBe(true);
    expect(result.errors).toHaveLength(0);
  });

  it('formats lib issue on second line with line:col', async () => {
    const dir = mkdtempSync(join(tmpdir(), 'cspell-'));
    writeFileSync(join(dir, 'cspell.json'), '{"version":"0.2","words":[]}');
    writeFileSync(join(dir, 'two.md'), 'first line\nzzzq second');
    const result = await cspellCheck.run(dir);
    expect(result.ok).toBe(false);
    expect(result.errors.some((e) => e.includes('zzzq') && e.includes('2:'))).toBe(true);
  });

  it('lib path catches when spellCheckFile throws', async () => {
    (globalThis as { __cspellThrowOnce?: boolean }).__cspellThrowOnce = true;
    const dir = mkdtempSync(join(tmpdir(), 'cspell-'));
    writeFileSync(join(dir, 'cspell.json'), '{"version":"0.2","words":[]}');
    writeFileSync(join(dir, 'a.md'), 'x');
    const result = await cspellCheck.run(dir);
    expect(result.ok).toBe(false);
    expect(result.errors.some((e) => e.includes('cspell reported issues'))).toBe(true);
  });

  it('lib path formats issue with no word (branch coverage)', async () => {
    (globalThis as { __cspellReturnNoTextIssue?: boolean }).__cspellReturnNoTextIssue = true;
    const dir = mkdtempSync(join(tmpdir(), 'cspell-'));
    writeFileSync(join(dir, 'cspell.json'), '{"version":"0.2","words":[]}');
    writeFileSync(join(dir, 'b.md'), 'x');
    const result = await cspellCheck.run(dir);
    expect(result.ok).toBe(false);
    expect(result.errors[0]).toMatch(/1:1 - Unknown$/);
  });

  it('CLI path parses Files checked from output', async () => {
    const dir = mkdtempSync(join(tmpdir(), 'cspell-'));
    writeFileSync(join(dir, 'cspell.json'), '{"version":"0.2","words":[]}');
    writeFileSync(join(dir, 'c.md'), 'ok');
    const result = await cspellCheck.run(dir, {
      _execSync: () => 'Files checked: 2\n',
    });
    expect(result.ok).toBe(true);
    expect(result.meta?.filesChecked).toBe(2);
  });

  it('with stagedFiles runs only on staged paths', async () => {
    const dir = mkdtempSync(join(tmpdir(), 'cspell-'));
    writeFileSync(join(dir, 'cspell.json'), '{"version":"0.2","words":[]}');
    writeFileSync(join(dir, 'staged.md'), 'x');
    const result = await cspellCheck.run(dir, {
      stagedFiles: ['staged.md'],
      _execSync: (cmd) => {
        if (cmd.includes('staged.md')) {
          const err = new Error() as Error & { status: number; stdout: string };
          err.status = 1;
          err.stdout = `${join(dir, 'staged.md')}:1:1 - Unknown word: xyzzyspoon`;
          throw err;
        }
        return '';
      },
    });
    expect(result.ok).toBe(false);
    expect(result.errors.some((e) => e.includes('xyzzyspoon') || e.includes('staged.md'))).toBe(true);
  });

  it('with stagedFiles mock return path when cmd does not match', async () => {
    const dir = mkdtempSync(join(tmpdir(), 'cspell-'));
    writeFileSync(join(dir, 'cspell.json'), '{"version":"0.2","words":[]}');
    writeFileSync(join(dir, 'other.md'), 'valid text');
    const result = await cspellCheck.run(dir, {
      stagedFiles: ['other.md'],
      _execSync: (cmd) => {
        if (cmd.includes('staged.md')) throw new Error();
        return '';
      },
    });
    expect(result.ok).toBe(true);
  });

  it('passes when stagedFiles listed but none exist (nothing to check)', async () => {
    const dir = mkdtempSync(join(tmpdir(), 'cspell-'));
    writeFileSync(join(dir, 'cspell.json'), '{"version":"0.2","words":[]}');
    writeFileSync(join(dir, 'package.json'), '{"devDependencies":{"cspell":"^8.0.0"}}');
    const result = await cspellCheck.run(dir, { stagedFiles: ['nonexistent.md'] });
    expect(result.ok).toBe(true);
    expect(result.meta?.filesChecked).toBe(0);
  });

  it('CLI path with staged but none exist returns 0 files', async () => {
    const dir = mkdtempSync(join(tmpdir(), 'cspell-'));
    writeFileSync(join(dir, 'cspell.json'), '{"version":"0.2","words":[]}');
    const result = await cspellCheck.run(dir, {
      stagedFiles: ['nonexistent.md'],
      _execSync: () => '',
    });
    expect(result.ok).toBe(true);
    expect(result.meta?.filesChecked).toBe(0);
  });

  it('returns fallback error when exec exits non-zero and output has no parseable issues', async () => {
    const dir = mkdtempSync(join(tmpdir(), 'cspell-'));
    writeFileSync(join(dir, 'cspell.json'), '{"version":"0.2","words":[]}');
    writeFileSync(join(dir, 'package.json'), '{"devDependencies":{"cspell":"^8.0.0"}}');
    writeFileSync(join(dir, 'doc.md'), 'x');
    const err = new Error() as Error & { status: number; stdout?: string; stderr?: string };
    err.status = 1;
    err.stdout = '';
    err.stderr = '';
    const result = await cspellCheck.run(dir, {
      _execSync: () => {
        throw err;
      },
    });
    expect(result.ok).toBe(false);
    expect(result.errors[0]).toContain('cspell reported issues');
  });

  it('uses exit code 1 when thrown error has no status (glob path)', async () => {
    const dir = mkdtempSync(join(tmpdir(), 'cspell-'));
    writeFileSync(join(dir, 'cspell.json'), '{"version":"0.2","words":[]}');
    writeFileSync(join(dir, 'package.json'), '{"devDependencies":{"cspell":"^8.0.0"}}');
    writeFileSync(join(dir, 'doc.md'), 'x');
    const result = await cspellCheck.run(dir, {
      _execSync: () => {
        throw new Error('no status');
      },
    });
    expect(result.ok).toBe(false);
  });

  it('uses exit code 1 when thrown error has no status (staged path)', async () => {
    const dir = mkdtempSync(join(tmpdir(), 'cspell-'));
    writeFileSync(join(dir, 'cspell.json'), '{"version":"0.2","words":[]}');
    writeFileSync(join(dir, 'package.json'), '{"devDependencies":{"cspell":"^8.0.0"}}');
    writeFileSync(join(dir, 's.md'), 'x');
    const result = await cspellCheck.run(dir, {
      stagedFiles: ['s.md'],
      _execSync: () => {
        throw new Error('no status');
      },
    });
    expect(result.ok).toBe(false);
  });
});

describe('runCspell', () => {
  it('returns empty output and 0 when paths is empty', () => {
    const dir = mkdtempSync(join(tmpdir(), 'cspell-'));
    const result = runCspell(dir, []);
    expect(result.output).toBe('');
    expect(result.exitCode).toBe(0);
  });
});

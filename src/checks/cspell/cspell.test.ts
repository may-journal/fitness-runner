import { mkdtempSync, writeFileSync } from 'node:fs';
import { join } from 'node:path';
import { tmpdir } from 'node:os';
import { describe, it, expect, vi } from 'vitest';
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

  it('with stagedFiles runs only on staged paths', async () => {
    const dir = mkdtempSync(join(tmpdir(), 'cspell-'));
    writeFileSync(join(dir, 'cspell.json'), '{"version":"0.2","words":[]}');
    writeFileSync(join(dir, 'package.json'), '{"devDependencies":{"cspell":"^8.0.0"}}');
    writeFileSync(join(dir, 'staged.md'), 'teh');
    const result = await cspellCheck.run(dir, { stagedFiles: ['staged.md'] });
    expect(result.ok).toBe(false);
  });

  it('passes when stagedFiles listed but none exist (nothing to check)', async () => {
    const dir = mkdtempSync(join(tmpdir(), 'cspell-'));
    writeFileSync(join(dir, 'cspell.json'), '{"version":"0.2","words":[]}');
    writeFileSync(join(dir, 'package.json'), '{"devDependencies":{"cspell":"^8.0.0"}}');
    const result = await cspellCheck.run(dir, { stagedFiles: ['nonexistent.md'] });
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

import { mkdtempSync, writeFileSync } from 'node:fs';
import { join } from 'node:path';
import { tmpdir } from 'node:os';
import { beforeEach, describe, it, expect, vi } from 'vitest';
import { changelogUpdatedCheck } from './index.js';

vi.mock('node:child_process', async (importOriginal) => {
  const mod = await importOriginal<typeof import('node:child_process')>();
  return { ...mod, execSync: vi.fn(mod.execSync) };
});

describe('changelogUpdatedCheck', () => {
  let dir: string;
  beforeEach(async () => {
    dir = mkdtempSync(join(tmpdir(), 'changelog-updated-'));
    const { execSync } = await import('node:child_process');
    vi.mocked(execSync).mockReset();
  });

  it('passes when no staged context', async () => {
    writeFileSync(join(dir, 'CHANGELOG.md'), '# Changelog\n\n### 2026-02-15\n\n- item');
    expect(await changelogUpdatedCheck.run(dir)).toMatchObject({ ok: true, meta: { filesChecked: 0 } });
    expect(await changelogUpdatedCheck.run(dir, { stagedFiles: [] })).toMatchObject({
      ok: true,
      meta: { filesChecked: 0 },
    });
  });

  it('passes when staged diff has no significant words', async () => {
    const { execSync } = await import('node:child_process');
    vi.mocked(execSync).mockReturnValueOnce('+++ a\n+ x\n+ y\n');
    writeFileSync(join(dir, 'CHANGELOG.md'), '# Changelog\n\n### 2026-02-15\n\n- item');
    const result = await changelogUpdatedCheck.run(dir, { stagedFiles: ['foo.ts'] });
    expect(result.ok).toBe(true);
    expect(result.meta?.filesChecked).toBe(1);
  });

  it('passes when changelog contains at least three words from staged diff', async () => {
    const { execSync } = await import('node:child_process');
    vi.mocked(execSync).mockReturnValueOnce(
      '+++ src/foo.ts\n+ Added new runner feature for validation.\n+ Export runner from index.\n',
    );
    writeFileSync(
      join(dir, 'CHANGELOG.md'),
      '# Changelog\n\n### 2026-02-15\n\n- Added new runner feature; validation export.',
    );
    const result = await changelogUpdatedCheck.run(dir, { stagedFiles: ['src/foo.ts'] });
    expect(result.ok).toBe(true);
    expect(result.meta?.filesChecked).toBe(1);
  });

  it('fails when CHANGELOG.md missing with staged files', async () => {
    const { execSync } = await import('node:child_process');
    vi.mocked(execSync).mockReturnValueOnce('+ feature code\n');
    const result = await changelogUpdatedCheck.run(dir, { stagedFiles: ['src/bar.ts'] });
    expect(result.ok).toBe(false);
    expect(result.errors?.[0]).toContain('CHANGELOG.md missing');
  });

  it('fails when fewer than three overlapping words', async () => {
    const { execSync } = await import('node:child_process');
    vi.mocked(execSync).mockReturnValueOnce(
      '+++ src/baz.ts\n+ New feature runner validation export helper.\n',
    );
    writeFileSync(join(dir, 'CHANGELOG.md'), '# Changelog\n\n### 2026-02-15\n\n- Minor fix.');
    const result = await changelogUpdatedCheck.run(dir, { stagedFiles: ['src/baz.ts'] });
    expect(result.ok).toBe(false);
    expect(result.errors?.[0]).toMatch(/at least 3 words/);
    expect(result.errors?.[0]).toMatch(/found \d+/);
  });
});

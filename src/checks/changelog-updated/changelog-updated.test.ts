import { mkdtempSync, writeFileSync } from 'node:fs';
import { join } from 'node:path';
import { tmpdir } from 'node:os';
import { execSync } from 'node:child_process';
import { beforeEach, describe, it, expect, vi } from 'vitest';
import {
  changelogUpdatedCheck,
  MSG_CHANGELOG_MISSING,
  MSG_CHANGELOG_TIME,
  MSG_OVERLAP_HEAD,
  MSG_OVERLAP_TAIL,
  MSG_STAGE_CHANGELOG,
  MSG_SUGGEST_PREFIX,
} from './index.js';

vi.mock('node:child_process', async (importOriginal) => {
  const mod = await importOriginal<typeof import('node:child_process')>();
  return { ...mod, execSync: vi.fn(mod.execSync) };
});

describe('changelogUpdatedCheck', () => {
  let dir: string;
  beforeEach(() => {
    dir = mkdtempSync(join(tmpdir(), 'changelog-updated-'));
    vi.mocked(execSync).mockReset();
  });

  it('passes when no staged context', async () => {
    writeFileSync(join(dir, 'CHANGELOG.md'), '# Changelog\n\n### 2026-02-15\n\n- item');
    expect(await changelogUpdatedCheck.run(dir)).toMatchObject({
      ok: true,
      meta: { filesChecked: 0 },
    });
    expect(await changelogUpdatedCheck.run(dir, { stagedFiles: [] })).toMatchObject({
      ok: true,
      meta: { filesChecked: 0 },
    });
  });

  it('passes when only CHANGELOG.md has staged additions (no other files to compare)', async () => {
    const mockExec = () => '+++ b/CHANGELOG.md\n+ - item\n';
    writeFileSync(join(dir, 'CHANGELOG.md'), '# Changelog\n\n### 2026-02-15\n\n- item');
    const result = await changelogUpdatedCheck.run(dir, {
      stagedFiles: ['CHANGELOG.md'],
      _execSync: mockExec,
    });
    expect(result.ok).toBe(true);
    expect(result.meta?.filesChecked).toBe(1);
  });

  it('passes when changelog additions share at least three words with rest of staged diff', async () => {
    const mockExec = () =>
      '+++ b/src/foo.ts\n+ Added new runner feature for validation.\n+ Export runner from index.\n+++ b/CHANGELOG.md\n+ - Added new runner feature; validation export.\n';
    writeFileSync(join(dir, 'CHANGELOG.md'), '# Changelog\n\n### 2026-02-15\n\n- item');
    const result = await changelogUpdatedCheck.run(dir, {
      stagedFiles: ['src/foo.ts', 'CHANGELOG.md'],
      _execSync: mockExec,
    });
    expect(result.ok).toBe(true);
    expect(result.meta?.filesChecked).toBe(1);
  });

  it('uses execSync when _execSync not in context', async () => {
    vi.mocked(execSync).mockReturnValue(
      '+++ b/src/foo.ts\n+ Added new runner feature.\n+++ b/CHANGELOG.md\n+ - Added new runner feature.\n'
    );
    writeFileSync(join(dir, 'CHANGELOG.md'), '# Changelog\n\n### 2026-02-15\n\n- item');
    const result = await changelogUpdatedCheck.run(dir, {
      stagedFiles: ['src/foo.ts', 'CHANGELOG.md'],
    });
    expect(result.ok).toBe(true);
    expect(vi.mocked(execSync).mock.calls[0][0]).toBe('git diff --cached');
  });

  it('fails when CHANGELOG.md missing with staged files', async () => {
    const mockExec = () => '+ feature code\n';
    const result = await changelogUpdatedCheck.run(dir, {
      stagedFiles: ['src/bar.ts'],
      _execSync: mockExec,
    });
    expect(result.ok).toBe(false);
    expect(result.errors?.[0]).toBe(MSG_CHANGELOG_MISSING);
  });

  it('fails when changelog additions share fewer than three words with rest of diff', async () => {
    const mockExec = () =>
      '+++ b/src/baz.ts\n+ New feature runner validation export helper.\n+++ b/CHANGELOG.md\n+ - Minor fix.\n';
    writeFileSync(join(dir, 'CHANGELOG.md'), '# Changelog\n\n### 2026-02-15\n\n- Minor fix.');
    const result = await changelogUpdatedCheck.run(dir, {
      stagedFiles: ['src/baz.ts', 'CHANGELOG.md'],
      _execSync: mockExec,
    });
    expect(result.ok).toBe(false);
    expect(result.errors?.[0]).toContain(MSG_OVERLAP_HEAD);
    expect(result.errors?.[0]).toContain(MSG_OVERLAP_TAIL);
    expect(result.errors?.[0]).toMatch(/found \d+/);
    expect(result.errors).toHaveLength(2);
    expect(result.errors?.[1]).toContain(MSG_SUGGEST_PREFIX);
  });

  it('fails when CHANGELOG.md not staged (no additions in diff)', async () => {
    const mockExec = () => '+++ b/src/bar.ts\n+ New feature code here.\n';
    writeFileSync(join(dir, 'CHANGELOG.md'), '# Changelog\n\n### 2026-02-15\n\n- item');
    const result = await changelogUpdatedCheck.run(dir, {
      stagedFiles: ['src/bar.ts'],
      _execSync: mockExec,
    });
    expect(result.ok).toBe(false);
    expect(result.errors?.[0]).toBe(MSG_STAGE_CHANGELOG);
  });

  it('passes when new section heading uses current date and time', async () => {
    const fixed = '2026.02.16.1430';
    const mockExec = () =>
      `+++ b/src/foo.ts\n+ New runner feature.\n+++ b/CHANGELOG.md\n+ ### ${fixed}\n+ - New runner feature.\n`;
    writeFileSync(join(dir, 'CHANGELOG.md'), '# Changelog\n\n### 2026.02.15\n\n- item');
    const result = await changelogUpdatedCheck.run(dir, {
      stagedFiles: ['src/foo.ts', 'CHANGELOG.md'],
      _execSync: mockExec,
      _now: () => new Date(2026, 1, 16, 14, 30),
    });
    expect(result.ok).toBe(true);
    expect(result.meta?.filesChecked).toBe(1);
  });

  it('fails when new section heading does not match current time', async () => {
    const mockExec = () =>
      '+++ b/src/foo.ts\n+ New runner feature.\n+++ b/CHANGELOG.md\n+ ### 2026.02.16.1900\n+ - New runner feature.\n';
    writeFileSync(join(dir, 'CHANGELOG.md'), '# Changelog\n\n### 2026.02.15\n\n- item');
    const result = await changelogUpdatedCheck.run(dir, {
      stagedFiles: ['src/foo.ts', 'CHANGELOG.md'],
      _execSync: mockExec,
      _now: () => new Date(2026, 1, 16, 14, 30),
    });
    expect(result.ok).toBe(false);
    expect(result.errors?.[0]).toContain(MSG_CHANGELOG_TIME);
    expect(result.errors?.[0]).toContain('expected ### 2026.02.16.1430');
  });
});

import { mkdtempSync } from 'node:fs';
import { join } from 'node:path';
import { tmpdir } from 'node:os';
import { describe, it, expect, vi } from 'vitest';
import { MSG_EMPTY, semanticCheck } from './index.js';

vi.mock('node:child_process', async (importOriginal) => {
  const mod = await importOriginal<typeof import('node:child_process')>();
  return { ...mod, execSync: vi.fn(mod.execSync) };
});

describe('semanticCheck', () => {
  it('passes when HEAD follows type(scope): description', async () => {
    const { execSync } = await import('node:child_process');
    vi.mocked(execSync).mockImplementationOnce(() => 'feat(api): add endpoint\n\nBody');
    const dir = mkdtempSync(join(tmpdir(), 'semantic-'));
    const result = await semanticCheck.run(dir);
    expect(result.ok).toBe(true);
  });

  it('passes for valid semantic format from mock', async () => {
    const { execSync } = await import('node:child_process');
    vi.mocked(execSync).mockImplementationOnce(() => 'feat(api): add endpoint\n\nBody');
    const dir = mkdtempSync(join(tmpdir(), 'semantic-'));
    const result = await semanticCheck.run(dir);
    expect(result.ok).toBe(true);
  });

  it('fails when HEAD does not follow format', async () => {
    const { execSync } = await import('node:child_process');
    vi.mocked(execSync).mockImplementationOnce(() => 'Fix something\n\nBody');
    const dir = mkdtempSync(join(tmpdir(), 'semantic-'));
    const result = await semanticCheck.run(dir);
    expect(result.ok).toBe(false);
    expect(result.errors[0]).toContain('type(scope): description');
  });

  it('passes for merge commits', async () => {
    const { execSync } = await import('node:child_process');
    vi.mocked(execSync).mockImplementationOnce(() => "Merge branch 'feature' into main");
    const dir = mkdtempSync(join(tmpdir(), 'semantic-'));
    const result = await semanticCheck.run(dir);
    expect(result.ok).toBe(true);
  });

  it('fails when git fails (no repo / nothing to check)', async () => {
    const { execSync } = await import('node:child_process');
    vi.mocked(execSync).mockImplementationOnce(() => {
      throw new Error('not a git repo');
    });
    const dir = mkdtempSync(join(tmpdir(), 'semantic-'));
    const result = await semanticCheck.run(dir);
    expect(result.ok).toBe(false);
    expect(result.errors).toContain(MSG_EMPTY);
  });

  it('validates proposedCommitMessage from context (commit-msg hook)', async () => {
    const dir = mkdtempSync(join(tmpdir(), 'semantic-'));
    const pass = await semanticCheck.run(dir, { proposedCommitMessage: 'feat(api): add endpoint' });
    expect(pass.ok).toBe(true);
    const fail = await semanticCheck.run(dir, { proposedCommitMessage: 'wip stuff' });
    expect(fail.ok).toBe(false);
    expect(fail.errors?.[0]).toContain('Commit message:');
  });

  it('fails when git log returns no first line (nothing to check)', async () => {
    const { execSync } = await import('node:child_process');
    vi.mocked(execSync).mockImplementationOnce(() => ({ split: () => [] }) as unknown as string);
    const dir = mkdtempSync(join(tmpdir(), 'semantic-'));
    const result = await semanticCheck.run(dir);
    expect(result.ok).toBe(false);
    expect(result.errors).toContain(MSG_EMPTY);
  });

  it('fails when proposedCommitMessage is empty', async () => {
    const dir = mkdtempSync(join(tmpdir(), 'semantic-'));
    const result = await semanticCheck.run(dir, { proposedCommitMessage: '' });
    expect(result.ok).toBe(false);
    expect(result.errors).toContain(MSG_EMPTY);
  });
});

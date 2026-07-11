import { mkdtempSync } from 'node:fs';
import { join } from 'node:path';
import { tmpdir } from 'node:os';
import { describe, it, expect, vi } from 'vitest';
import {
  commitAttributionCheck,
  hasTrailer,
  isExemptSubject,
  missingTrailers,
  MSG_EMPTY,
  REQUIRED_TRAILERS,
} from './index.js';

vi.mock('node:child_process', async (importOriginal) => {
  const mod = await importOriginal<typeof import('node:child_process')>();
  return { ...mod, execSync: vi.fn(mod.execSync) };
});

const VALID = 'feat(api): add endpoint\n\nBody\n\nAI-Tools: Claude Code\nAI-Models: Opus 4.8';

describe('commitAttributionCheck', () => {
  it('passes when HEAD has both AI-Tools and AI-Models trailers', async () => {
    const { execSync } = await import('node:child_process');
    vi.mocked(execSync).mockImplementationOnce(() => VALID);
    const dir = mkdtempSync(join(tmpdir(), 'attr-'));
    const result = await commitAttributionCheck.run(dir);
    expect(result.ok).toBe(true);
    expect(result.meta.filesChecked).toBe(1);
  });

  it('fails when the AI-Tools trailer is missing', async () => {
    const { execSync } = await import('node:child_process');
    vi.mocked(execSync).mockImplementationOnce(
      () => 'feat(api): add endpoint\n\nAI-Models: Opus 4.8'
    );
    const dir = mkdtempSync(join(tmpdir(), 'attr-'));
    const result = await commitAttributionCheck.run(dir);
    expect(result.ok).toBe(false);
    expect(result.errors).toContain('commit message missing "AI-Tools:" trailer');
    expect(result.errors).not.toContain('commit message missing "AI-Models:" trailer');
  });

  it('fails when the AI-Models trailer is missing', async () => {
    const { execSync } = await import('node:child_process');
    vi.mocked(execSync).mockImplementationOnce(
      () => 'feat(api): add endpoint\n\nAI-Tools: Claude Code'
    );
    const dir = mkdtempSync(join(tmpdir(), 'attr-'));
    const result = await commitAttributionCheck.run(dir);
    expect(result.ok).toBe(false);
    expect(result.errors).toContain('commit message missing "AI-Models:" trailer');
  });

  it('fails naming both trailers when neither is present', async () => {
    const { execSync } = await import('node:child_process');
    vi.mocked(execSync).mockImplementationOnce(() => 'feat(api): add endpoint\n\nJust a body');
    const dir = mkdtempSync(join(tmpdir(), 'attr-'));
    const result = await commitAttributionCheck.run(dir);
    expect(result.ok).toBe(false);
    expect(result.errors).toHaveLength(2);
  });

  it('exempts merge commits', async () => {
    const { execSync } = await import('node:child_process');
    vi.mocked(execSync).mockImplementationOnce(() => "Merge branch 'feature' into main");
    const dir = mkdtempSync(join(tmpdir(), 'attr-'));
    const result = await commitAttributionCheck.run(dir);
    expect(result.ok).toBe(true);
  });

  it('exempts revert commits', async () => {
    const { execSync } = await import('node:child_process');
    vi.mocked(execSync).mockImplementationOnce(() => 'Revert "feat(api): add endpoint"');
    const dir = mkdtempSync(join(tmpdir(), 'attr-'));
    const result = await commitAttributionCheck.run(dir);
    expect(result.ok).toBe(true);
  });

  it('fails when git fails (no repo / nothing to check)', async () => {
    const { execSync } = await import('node:child_process');
    vi.mocked(execSync).mockImplementationOnce(() => {
      throw new Error('not a git repo');
    });
    const dir = mkdtempSync(join(tmpdir(), 'attr-'));
    const result = await commitAttributionCheck.run(dir);
    expect(result.ok).toBe(false);
    expect(result.errors).toContain(MSG_EMPTY);
  });

  it('fails when git log returns no first line (nothing to check)', async () => {
    const { execSync } = await import('node:child_process');
    vi.mocked(execSync).mockImplementationOnce(() => ({ split: () => [] }) as unknown as string);
    const dir = mkdtempSync(join(tmpdir(), 'attr-'));
    const result = await commitAttributionCheck.run(dir);
    expect(result.ok).toBe(false);
    expect(result.errors).toContain(MSG_EMPTY);
  });

  it('validates proposedCommitMessage from context (commit-msg hook)', async () => {
    const dir = mkdtempSync(join(tmpdir(), 'attr-'));
    const pass = await commitAttributionCheck.run(dir, { proposedCommitMessage: VALID });
    expect(pass.ok).toBe(true);
    const fail = await commitAttributionCheck.run(dir, {
      proposedCommitMessage: 'feat(api): add endpoint\n\nno trailers',
    });
    expect(fail.ok).toBe(false);
    expect(fail.errors).toHaveLength(2);
  });

  it('fails when proposedCommitMessage is empty', async () => {
    const dir = mkdtempSync(join(tmpdir(), 'attr-'));
    const result = await commitAttributionCheck.run(dir, { proposedCommitMessage: '' });
    expect(result.ok).toBe(false);
    expect(result.errors).toContain(MSG_EMPTY);
  });
});

describe('helpers', () => {
  it('isExemptSubject flags merge and revert only', () => {
    expect(isExemptSubject('Merge branch x')).toBe(true);
    expect(isExemptSubject('Revert "x"')).toBe(true);
    expect(isExemptSubject('feat(api): add endpoint')).toBe(false);
  });

  it('hasTrailer requires a non-empty value on its own line', () => {
    expect(hasTrailer('subject\n\nAI-Tools: Claude Code', 'AI-Tools')).toBe(true);
    expect(hasTrailer('subject\n\nAI-Tools:', 'AI-Tools')).toBe(false);
    expect(hasTrailer('subject\n\nAI-Tools:   ', 'AI-Tools')).toBe(false);
    expect(hasTrailer('subject\n\nno trailer here', 'AI-Tools')).toBe(false);
  });

  it('missingTrailers reports each absent trailer', () => {
    expect(missingTrailers(VALID)).toEqual([]);
    expect(missingTrailers('subject\n\nAI-Tools: x')).toEqual([
      'commit message missing "AI-Models:" trailer',
    ]);
    expect(REQUIRED_TRAILERS).toContain('AI-Tools');
    expect(REQUIRED_TRAILERS).toContain('AI-Models');
  });
});

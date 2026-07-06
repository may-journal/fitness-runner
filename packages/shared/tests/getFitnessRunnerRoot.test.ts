import { mkdtempSync } from 'node:fs';
import { join } from 'node:path';
import { tmpdir } from 'node:os';
import { existsSync } from 'node:fs';
import { describe, expect, it, vi } from 'vitest';

describe('getFitnessRunnerRoot', () => {
  it('returns the bundled config directory containing cspell.json', async () => {
    const { getFitnessRunnerRoot } = await import('@mayjournal/fitness-shared');
    const root = getFitnessRunnerRoot();
    expect(existsSync(join(root, 'cspell.json'))).toBe(true);
    expect(existsSync(join(root, 'eslint.config.mjs'))).toBe(true);
  });

  it('finds bundled configs when cwd is not the repo root', async () => {
    const { getFitnessRunnerRoot } = await import('@mayjournal/fitness-shared');
    const temp = mkdtempSync(join(tmpdir(), 'fitness-root-'));
    vi.spyOn(process, 'cwd').mockReturnValue(temp);
    try {
      const root = getFitnessRunnerRoot();
      expect(existsSync(join(root, 'cspell.json'))).toBe(true);
    } finally {
      vi.restoreAllMocks();
    }
  });
});

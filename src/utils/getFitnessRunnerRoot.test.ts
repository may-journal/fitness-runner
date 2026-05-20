import { existsSync } from 'node:fs';
import { join } from 'node:path';
import { describe, expect, it } from 'vitest';

describe('getFitnessRunnerRoot', () => {
  it('returns the shared config directory containing cspell.json', async () => {
    const { getFitnessRunnerRoot } = await import('./getFitnessRunnerRoot.js');
    const root = getFitnessRunnerRoot();
    expect(existsSync(join(root, 'cspell.json'))).toBe(true);
    expect(existsSync(join(root, 'eslint.config.cjs'))).toBe(true);
  });
});

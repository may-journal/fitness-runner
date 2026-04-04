import { parse } from 'node:path';
import { existsSync } from 'node:fs';
import { join } from 'node:path';
import { afterEach, describe, expect, it, vi } from 'vitest';

describe('getFitnessRunnerRoot', () => {
  afterEach(() => {
    vi.resetModules();
    vi.doUnmock('node:fs');
  });

  it('returns a directory containing package.json', async () => {
    const { getFitnessRunnerRoot } = await import('./getFitnessRunnerRoot.js');
    const root = getFitnessRunnerRoot();
    expect(existsSync(join(root, 'package.json'))).toBe(true);
  });

  it('returns filesystem root when existsSync never finds package.json', async () => {
    vi.doMock('node:fs', () => ({ existsSync: () => false }));
    vi.resetModules();
    const { getFitnessRunnerRoot } = await import('./getFitnessRunnerRoot.js');
    expect(getFitnessRunnerRoot()).toBe(parse(process.cwd()).root);
  });
});

import { mkdtempSync, writeFileSync } from 'node:fs';
import { join } from 'node:path';
import { tmpdir } from 'node:os';
import { describe, it, expect } from 'vitest';
import { getFitnessRunnerRoot } from './getFitnessRunnerRoot.js';
import { resolveFitnessConfigPath } from './resolveFitnessConfigPath.js';

describe('resolveFitnessConfigPath', () => {
  it('returns local path when the consumer has the config file', () => {
    const dir = mkdtempSync(join(tmpdir(), 'resolve-config-'));
    writeFileSync(join(dir, 'cspell.json'), '{}');
    expect(resolveFitnessConfigPath(dir, 'cspell.json')).toBe(join(dir, 'cspell.json'));
  });

  it('returns package path when the consumer has no local config', () => {
    const dir = mkdtempSync(join(tmpdir(), 'resolve-config-'));
    const frRoot = getFitnessRunnerRoot();
    expect(resolveFitnessConfigPath(dir, 'cspell.json')).toBe(join(frRoot, 'cspell.json'));
  });
});

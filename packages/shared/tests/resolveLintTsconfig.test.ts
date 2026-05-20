import { mkdtempSync, readFileSync } from 'node:fs';
import { join } from 'node:path';
import { tmpdir } from 'node:os';
import { describe, it, expect } from 'vitest';
import { getFitnessRunnerRoot, resolveLintTsconfig } from '@mayjournal/fitness-shared';

describe('resolveLintTsconfig', () => {
  it('writes a temp tsconfig with absolute include for the consumer root', () => {
    const dir = mkdtempSync(join(tmpdir(), 'lint-tsconfig-'));
    const frRoot = getFitnessRunnerRoot();
    const path = resolveLintTsconfig(dir, frRoot);
    const config = JSON.parse(readFileSync(path, 'utf8')) as { include: string[] };
    expect(config.include[0]).toBe(join(dir, '**/*.ts'));
  });

  it('returns the same path on repeated calls for the same root', () => {
    const dir = mkdtempSync(join(tmpdir(), 'lint-tsconfig-'));
    const frRoot = getFitnessRunnerRoot();
    expect(resolveLintTsconfig(dir, frRoot)).toBe(resolveLintTsconfig(dir, frRoot));
  });
});

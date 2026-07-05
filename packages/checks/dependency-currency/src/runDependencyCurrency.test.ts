import { mkdtempSync, writeFileSync } from 'node:fs';
import { join } from 'node:path';
import { tmpdir } from 'node:os';
import { describe, it, expect } from 'vitest';
import { dependencyCurrencyCheck, enUS, outdatedErrors, parseOutdated } from './index.js';

/** Temp project dir with a single package.json, so countManifests reports 1. */
function projectDir(): string {
  const dir = mkdtempSync(join(tmpdir(), 'dep-currency-'));
  writeFileSync(join(dir, 'package.json'), '{"name":"tmp","version":"1.0.0"}');
  return dir;
}

/** Simulates npm exiting 1 (deps are behind) with JSON on stdout. */
function exits1With(stdout: string): () => never {
  return () => {
    throw { status: 1, stdout };
  };
}

describe('parseOutdated', () => {
  it('returns {} for empty output (nothing outdated)', () => {
    expect(parseOutdated('')).toEqual({});
    expect(parseOutdated('  \n')).toEqual({});
  });

  it('parses a JSON object', () => {
    expect(parseOutdated('{"chalk":{"current":"4.1.2","latest":"5.3.0"}}')).toEqual({
      chalk: { current: '4.1.2', latest: '5.3.0' },
    });
  });

  it('returns null when output is not a JSON object (e.g. registry error)', () => {
    expect(parseOutdated('npm error code ENOTFOUND')).toBeNull();
    expect(parseOutdated('null')).toBeNull();
  });
});

describe('outdatedErrors', () => {
  it('formats name: current → latest, sorted', () => {
    expect(
      outdatedErrors({
        zod: { current: '3.0.0', latest: '3.23.0' },
        chalk: { current: '4.1.2', latest: '5.3.0' },
      })
    ).toEqual(['chalk: 4.1.2 → 5.3.0', 'zod: 3.0.0 → 3.23.0']);
  });

  it('skips internal @mayjournal/* workspace packages', () => {
    expect(
      outdatedErrors({ '@mayjournal/fitness-shared': { current: '1.0.0', latest: '2.0.0' } })
    ).toEqual([]);
  });

  it('reports a missing (declared but not installed) dependency', () => {
    expect(outdatedErrors({ foo: { latest: '2.0.0' } })).toEqual(['foo: missing → 2.0.0']);
  });

  it('handles the array form (outdated in several locations)', () => {
    expect(
      outdatedErrors({
        vitest: [
          { current: '3.0.0', latest: '4.0.18' },
          { current: '2.0.0', latest: '4.0.18' },
        ],
      })
    ).toEqual(['vitest: 2.0.0 → 4.0.18', 'vitest: 3.0.0 → 4.0.18']);
  });

  it('ignores records already at latest', () => {
    expect(outdatedErrors({ chalk: { current: '5.3.0', latest: '5.3.0' } })).toEqual([]);
  });

  it('collapses identical lines (same dep+version across workspaces) to one', () => {
    expect(
      outdatedErrors({
        '@types/node': [
          { current: '25.3.0', latest: '26.1.0', dependent: 'a' },
          { current: '25.3.0', latest: '26.1.0', dependent: 'b' },
        ],
      })
    ).toEqual(['@types/node: 25.3.0 → 26.1.0']);
  });
});

describe('dependencyCurrencyCheck', () => {
  it('default export matches the check', async () => {
    const mod = await import('./index.js');
    expect(mod.default).toBe(mod.dependencyCurrencyCheck);
  });

  it('exposes enUS.Lead', () => {
    expect(enUS.Lead).toMatch(/behind their latest/);
  });

  it('has name dependency-currency', () => {
    expect(dependencyCurrencyCheck.name).toBe('dependency-currency');
  });

  it('passes when nothing is outdated', async () => {
    const result = await dependencyCurrencyCheck.run(projectDir(), { _execSync: () => '{}' });
    expect(result.ok).toBe(true);
    expect(result.errors).toHaveLength(0);
    expect(result.meta?.filesChecked).toBe(1);
  });

  it('fails and lists deps that are behind latest', async () => {
    const stdout = JSON.stringify({
      chalk: { current: '4.1.2', wanted: '4.1.2', latest: '5.3.0', dependent: 'tmp' },
    });
    const result = await dependencyCurrencyCheck.run(projectDir(), {
      _execSync: exits1With(stdout),
    });
    expect(result.ok).toBe(false);
    expect(result.errors).toContain('chalk: 4.1.2 → 5.3.0');
    expect(result.errors[0]).toBe(enUS.Lead);
  });

  it('degrades to a pass when the registry is unreachable', async () => {
    const result = await dependencyCurrencyCheck.run(projectDir(), {
      _execSync: exits1With('npm error code ENOTFOUND'),
    });
    expect(result.ok).toBe(true);
    expect(result.errors).toHaveLength(0);
  });
});

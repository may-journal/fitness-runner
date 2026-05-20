import { mkdtempSync, writeFileSync } from 'node:fs';
import { join } from 'node:path';
import { tmpdir } from 'node:os';
import { describe, it, expect } from 'vitest';
import { loadConfig } from '@mayjournal/fitness-shared';

describe('loadConfig', () => {
  it('returns null when no config file exists', () => {
    const dir = mkdtempSync(join(tmpdir(), 'fitness-config-'));
    expect(loadConfig(dir)).toBe(null);
  });

  it('returns config when .fitnessrc.ts exists with default export', () => {
    const dir = mkdtempSync(join(tmpdir(), 'fitness-config-'));
    writeFileSync(
      join(dir, '.fitnessrc.ts'),
      'export default { checks: ["changelog", "semantic-commit"] };'
    );
    const config = loadConfig(dir);
    expect(config).not.toBe(null);
    expect(config?.checks).toEqual(['changelog', 'semantic-commit']);
  });

  it('returns null when default export is null', () => {
    const dir = mkdtempSync(join(tmpdir(), 'fitness-config-'));
    writeFileSync(join(dir, '.fitnessrc.ts'), 'export default null;');
    expect(loadConfig(dir)).toBe(null);
  });

  it('returns null when .fitnessrc.js exports null (getRawExport receives null)', () => {
    const dir = mkdtempSync(join(tmpdir(), 'fitness-config-'));
    writeFileSync(join(dir, '.fitnessrc.js'), 'module.exports = null;');
    expect(loadConfig(dir)).toBe(null);
  });

  it('returns null when .fitnessrc.ts is invalid', () => {
    const dir = mkdtempSync(join(tmpdir(), 'fitness-config-'));
    writeFileSync(join(dir, '.fitnessrc.ts'), 'syntax error {{{');
    expect(loadConfig(dir)).toBe(null);
  });

  it('returns config when .fitnessrc.js exists (no .ts)', () => {
    const dir = mkdtempSync(join(tmpdir(), 'fitness-config-'));
    writeFileSync(join(dir, '.fitnessrc.js'), 'module.exports = { checks: ["changelog"] };');
    const config = loadConfig(dir);
    expect(config).not.toBe(null);
    expect(config?.checks).toEqual(['changelog']);
  });

  it('returns module when default export is missing (namespace object)', () => {
    const dir = mkdtempSync(join(tmpdir(), 'fitness-config-'));
    writeFileSync(join(dir, '.fitnessrc.ts'), 'export const checks = ["changelog"];');
    const config = loadConfig(dir);
    expect(config).not.toBe(null);
    expect(config?.checks).toEqual(['changelog']);
  });

  it('returns null when default export is undefined', () => {
    const dir = mkdtempSync(join(tmpdir(), 'fitness-config-'));
    writeFileSync(join(dir, '.fitnessrc.ts'), 'export default undefined;');
    expect(loadConfig(dir)).toBe(null);
  });

  it('returns null when default export is object with default null', () => {
    const dir = mkdtempSync(join(tmpdir(), 'fitness-config-'));
    writeFileSync(join(dir, '.fitnessrc.ts'), 'export default { default: null };');
    expect(loadConfig(dir)).toBe(null);
  });

  it('parseConfigModule returns null when raw.default is null (branch line 29)', () => {
    const dir = mkdtempSync(join(tmpdir(), 'fitness-config-'));
    writeFileSync(join(dir, '.fitnessrc.js'), 'module.exports = { default: { default: null } };');
    expect(loadConfig(dir)).toBe(null);
  });
});

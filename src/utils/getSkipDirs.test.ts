import { mkdtempSync, writeFileSync } from 'node:fs';
import { join } from 'node:path';
import { tmpdir } from 'node:os';
import { describe, it, expect } from 'vitest';
import { getSkipDirs } from './getSkipDirs.js';

describe('getSkipDirs', () => {
  it('returns skip dirs from .fitnessrc.ts skipTheseDirectories when present', () => {
    const dir = mkdtempSync(join(tmpdir(), 'skip-dirs-'));
    writeFileSync(
      join(dir, '.fitnessrc.ts'),
      'export default { skipTheseDirectories: ["node_modules", "dist", "custom"] };'
    );
    const set = getSkipDirs(dir);
    expect(set).toEqual(new Set(['node_modules', 'dist', 'custom']));
  });

  it('filters non-strings from skipTheseDirectories', () => {
    const dir = mkdtempSync(join(tmpdir(), 'skip-dirs-'));
    writeFileSync(
      join(dir, '.fitnessrc.ts'),
      'export default { skipTheseDirectories: ["a", 1, null, "b"] };'
    );
    const set = getSkipDirs(dir);
    expect(set).toEqual(new Set(['a', 'b']));
  });

  it('falls back to cspell.json ignorePaths when no skipTheseDirectories', () => {
    const dir = mkdtempSync(join(tmpdir(), 'skip-dirs-'));
    writeFileSync(
      join(dir, 'cspell.json'),
      '{"ignorePaths":["node_modules","dist","**/*.test.ts"]}'
    );
    const set = getSkipDirs(dir);
    expect(set).toEqual(new Set(['node_modules', 'dist']));
  });

  it('returns empty set when no config and no cspell.json', () => {
    const dir = mkdtempSync(join(tmpdir(), 'skip-dirs-'));
    const set = getSkipDirs(dir);
    expect(set).toEqual(new Set());
  });
});

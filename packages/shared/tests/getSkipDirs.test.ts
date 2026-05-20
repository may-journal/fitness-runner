import { mkdtempSync, writeFileSync } from 'node:fs';
import { join } from 'node:path';
import { tmpdir } from 'node:os';
import { describe, it, expect } from 'vitest';
import { getSkipDirs, RUNNER_SKIP_DIRS } from '@mayjournal/fitness-shared';

describe('getSkipDirs', () => {
  it('returns runner skip dirs merged with skipTheseDirectories', () => {
    const dir = mkdtempSync(join(tmpdir(), 'skip-dirs-'));
    writeFileSync(
      join(dir, '.fitnessrc.ts'),
      'export default { skipTheseDirectories: ["node_modules", "dist", "custom"] };'
    );
    const set = getSkipDirs(dir);
    expect(set).toEqual(new Set([...RUNNER_SKIP_DIRS, 'custom']));
  });

  it('filters non-strings from skipTheseDirectories and merges with runner skip dirs', () => {
    const dir = mkdtempSync(join(tmpdir(), 'skip-dirs-'));
    writeFileSync(
      join(dir, '.fitnessrc.ts'),
      'export default { skipTheseDirectories: ["a", 1, null, "b"] };'
    );
    const set = getSkipDirs(dir);
    expect(set).toEqual(new Set([...RUNNER_SKIP_DIRS, 'a', 'b']));
  });

  it('returns runner skip dirs merged with cspell ignorePaths when no skipTheseDirectories', () => {
    const dir = mkdtempSync(join(tmpdir(), 'skip-dirs-'));
    writeFileSync(
      join(dir, 'cspell.json'),
      '{"ignorePaths":["node_modules","dist","**/*.test.ts"]}'
    );
    const set = getSkipDirs(dir);
    expect(set).toEqual(new Set(RUNNER_SKIP_DIRS));
  });

  it('returns runner skip dirs when no config and no cspell.json', () => {
    const dir = mkdtempSync(join(tmpdir(), 'skip-dirs-'));
    const set = getSkipDirs(dir);
    expect(set).toEqual(new Set(RUNNER_SKIP_DIRS));
  });
});

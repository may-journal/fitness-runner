import { existsSync } from 'node:fs';
import { join } from 'node:path';
import { describe, it, expect } from 'vitest';
import { getCspellPackageConfigDir } from './getCspellPackageConfigDir.js';

describe('getCspellPackageConfigDir', () => {
  it('returns a directory containing cspell.json', () => {
    const dir = getCspellPackageConfigDir();
    expect(existsSync(join(dir, 'cspell.json'))).toBe(true);
  });
});

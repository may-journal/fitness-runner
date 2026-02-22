import { readFileSync } from 'node:fs';
import { join } from 'node:path';
import { loadConfig } from '../config/load.js';

/** Dir names from cspell.json ignorePaths (entries with no / or *). */
function getSkipDirsFromCspell(root: string): Set<string> {
  try {
    const raw = readFileSync(join(root, 'cspell.json'), 'utf-8');
    const data = JSON.parse(raw) as { ignorePaths?: string[] };
    const paths = data.ignorePaths ?? [];
    const dirs = paths.filter((p) => typeof p === 'string' && !p.includes('/') && !p.includes('*'));
    return new Set(dirs);
  } catch {
    return new Set();
  }
}

/** Dir names to skip when walking: from fitness config skipTheseDirectories, else cspell.json ignorePaths. */
export function getSkipDirs(root: string): Set<string> {
  const config = loadConfig(root);
  if (config?.skipTheseDirectories && Array.isArray(config.skipTheseDirectories)) {
    const valid = config.skipTheseDirectories.filter((p) => typeof p === 'string');
    return new Set(valid);
  }
  return getSkipDirsFromCspell(root);
}

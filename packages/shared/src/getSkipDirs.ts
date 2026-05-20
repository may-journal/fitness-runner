import { readFileSync } from 'node:fs';
import { join } from 'node:path';
import { loadConfig } from './config/load.js';

/** Runner always skips these dir names when walking; merged with config/cspell. */
export const RUNNER_SKIP_DIRS = ['node_modules', 'dist', 'coverage', '.git', '.husky'];

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

/** Skip dirs for file walking only; avoids loadConfig/jiti so the walk never blocks. */
export function getSkipDirsForWalk(root: string): Set<string> {
  return new Set([...RUNNER_SKIP_DIRS, ...getSkipDirsFromCspell(root)]);
}

/** Dir names to skip when walking: runner defaults (node_modules, etc.) plus fitness config skipTheseDirectories or cspell.json ignorePaths. */
export function getSkipDirs(root: string): Set<string> {
  const fromConfig = (() => {
    const config = loadConfig(root);
    if (config?.skipTheseDirectories && Array.isArray(config.skipTheseDirectories)) {
      const valid = config.skipTheseDirectories.filter(
        (p: unknown): p is string => typeof p === 'string'
      );
      return new Set(valid);
    }
    return getSkipDirsFromCspell(root);
  })();
  return new Set([...RUNNER_SKIP_DIRS, ...fromConfig]);
}

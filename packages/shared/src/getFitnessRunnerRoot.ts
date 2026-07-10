import { existsSync } from 'node:fs';
import { dirname, join } from 'node:path';
import { fileURLToPath } from 'node:url';
import { CSPELL_JSON } from './fileNames.js';

const SHARED_PACKAGE = '@mayjournal/fitness-shared';

/** Walk up from start while predicate matches; returns last matching dir or start. */
function walkUpWhile(start: string, predicate: (dir: string) => boolean): string {
  let dir = start;
  let lastMatch = start;
  while (dir !== dirname(dir)) {
    if (predicate(dir)) lastMatch = dir;
    dir = dirname(dir);
  }
  return lastMatch;
}

/** Nearest ancestor of dir that contains cspell.json (consumer or bundled config dir). */
function findConfigRoot(dir: string): string {
  return walkUpWhile(dir, (d) => existsSync(join(d, CSPELL_JSON)));
}

/** Directory containing @mayjournal/fitness-shared config files. */
function resolveSharedConfigDir(): string | null {
  try {
    const file = fileURLToPath(import.meta.resolve(`${SHARED_PACKAGE}/cspell`));
    return dirname(file);
  } catch {
    /* v8 ignore next -- import.meta.resolve unavailable outside installed shared package */
    return null;
  }
}

/** Resolves the bundled fitness config directory (cspell, eslint, prettier, vitest, tsconfig). */
export function getFitnessRunnerRoot(): string {
  const configDir = resolveSharedConfigDir();
  if (configDir != null) return findConfigRoot(configDir);
  return findConfigRoot(process.cwd());
}

import { readdirSync, statSync } from 'node:fs';
import { join } from 'node:path';

const SKIP_DIRS = new Set([
  'node_modules',
  'dist',
  'coverage',
  '.git',
  '.husky',
]);

/** Process one dir entry: recurse or push .md path. */
function processEntry(
  name: string,
  dir: string,
  relDir: string,
  out: string[],
  walk: (d: string, r: string) => void,
): void {
  const abs = join(dir, name);
  const rel = relDir ? `${relDir}/${name}` : name;
  if (statSync(abs).isDirectory()) walk(abs, rel);
  else if (name.endsWith('.md')) out.push(rel);
}

/** Returns relative paths of .md files under root (recursive), excluding node_modules, dist, coverage, .git, .husky. */
export function findMd(root: string): string[] {
  const out: string[] = [];
  /** Recursively collects .md paths into out. */
  function walk(dir: string, relDir: string): void {
    for (const name of readdirSync(dir)) {
      if (SKIP_DIRS.has(name)) continue;
      processEntry(name, dir, relDir, out, walk);
    }
  }
  walk(root, '');
  return out.sort();
}

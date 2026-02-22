import { readdirSync, statSync } from 'node:fs';
import { join } from 'node:path';
import { getSkipDirs } from './getSkipDirs.js';

/** Process one dir entry: recurse or push matching path. */
function processEntry(
  name: string,
  dir: string,
  relDir: string,
  extension: string,
  out: string[],
  walk: (d: string, r: string) => void
): void {
  const abs = join(dir, name);
  const rel = relDir ? `${relDir}/${name}` : name;
  if (statSync(abs).isDirectory()) walk(abs, rel);
  else if (name.endsWith(extension)) out.push(rel);
}

/**
 * Returns relative paths of files under root with the given extension, excluding skip dirs from config.
 * @param root - Root directory to walk
 * @param extension - File extension to match (e.g. ".md")
 * @returns Sorted relative paths
 */
export function findFilesByExtension(root: string, extension: string): string[] {
  const skipDirs = getSkipDirs(root);
  const out: string[] = [];
  /** Recursively collects matching paths into out. */
  function walk(dir: string, relDir: string): void {
    for (const name of readdirSync(dir)) {
      if (skipDirs.has(name)) continue;
      processEntry(name, dir, relDir, extension, out, walk);
    }
  }
  walk(root, '');
  return out.sort();
}

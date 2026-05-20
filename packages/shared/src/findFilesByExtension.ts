import { readdir, stat } from 'node:fs/promises';
import { join } from 'node:path';
import { getSkipDirsForWalk } from './getSkipDirs.js';

/**
 * Returns relative paths of files under root with the given extension, excluding skip dirs.
 * Uses getSkipDirsForWalk (runner + cspell only) so the walk never blocks on loadConfig/jiti.
 */
export async function findFilesByExtension(root: string, extension: string): Promise<string[]> {
  const skipDirs = getSkipDirsForWalk(root);
  const out: string[] = [];

  /** Process one dir entry: recurse into dirs or collect matching files. */
  async function visit(dir: string, relDir: string, name: string): Promise<void> {
    const abs = join(dir, name);
    const rel = relDir ? `${relDir}/${name}` : name;
    const statResult = await stat(abs);
    if (statResult.isDirectory()) await walk(abs, rel);
    else if (name.endsWith(extension)) out.push(rel);
  }

  /** Recursively walk dir and push matching relative paths into out. */
  async function walk(dir: string, relDir: string): Promise<void> {
    const names = await readdir(dir);
    for (const name of names) {
      if (skipDirs.has(name)) continue;
      await visit(dir, relDir, name);
    }
  }
  await walk(root, '');
  return out.sort();
}

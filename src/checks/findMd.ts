import { readdirSync, statSync } from 'node:fs';
import { join } from 'node:path';

/** Returns relative paths of all .md files under root (recursive). */
export function findMd(root: string): string[] {
  const out: string[] = [];
  function walk(dir: string, relDir: string): void {
    for (const name of readdirSync(dir)) {
      const abs = join(dir, name);
      const rel = relDir ? `${relDir}/${name}` : name;
      if (statSync(abs).isDirectory()) walk(abs, rel);
      else if (name.endsWith('.md')) out.push(rel);
    }
  }
  walk(root, '');
  return out.sort();
}

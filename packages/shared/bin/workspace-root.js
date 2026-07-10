import { existsSync, readFileSync } from 'node:fs';
import { dirname, join } from 'node:path';
import { PACKAGE_JSON } from './constants.js';

/** Walk up from start until a package.json with workspaces is found. */
export function findWorkspaceRoot(start) {
  let dir = start;
  while (true) {
    const parent = dirname(dir);
    if (parent === dir) return start;
    const pkgPath = join(dir, PACKAGE_JSON);
    if (existsSync(pkgPath)) {
      const pkg = JSON.parse(readFileSync(pkgPath, 'utf8'));
      if (pkg.workspaces) return dir;
    }
    dir = parent;
  }
}

/** Read name from package.json in dir, if present. */
export function readPackageName(dir) {
  const pkgPath = join(dir, PACKAGE_JSON);
  if (!existsSync(pkgPath)) return undefined;
  return JSON.parse(readFileSync(pkgPath, 'utf8')).name;
}

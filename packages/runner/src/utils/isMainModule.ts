import { realpathSync } from 'node:fs';
import { resolve } from 'node:path';
import { fileURLToPath } from 'node:url';

/** True if suffix is empty or a single extension (e.g. .js). */
function isExtensionSuffix(suffix: string): boolean {
  return suffix === '' || /^\.[a-z0-9]+$/i.test(suffix);
}

/** Resolves path to real path (follows symlinks); returns null if path does not exist or is inaccessible. */
function resolveReal(path: string): string | null {
  try {
    return realpathSync(path);
  } catch {
    return null;
  }
}

/** True when modulePath equals argvPath or is argvPath plus a single file extension (e.g. node foo → foo.js). */
function matchesArgvPath(modulePath: string, argvPath: string): boolean {
  if (modulePath === argvPath) return true;
  const suffix = modulePath.startsWith(argvPath) ? modulePath.slice(argvPath.length) : null;
  return suffix !== null && isExtensionSuffix(suffix);
}

/** True if the module with the given import.meta.url is the Node process entry. argv[1] may be a symlink (e.g. npx fitness → .bin/fitness); resolve to real paths before comparing. See https://2ality.com/2022/07/nodejs-esm-main.html */
export function isMainModule(importMetaUrl: string): boolean {
  const argvPath = resolve(process.argv[1] ?? '');
  const modulePath = resolve(fileURLToPath(importMetaUrl));
  const argvReal = resolveReal(argvPath);
  const moduleReal = resolveReal(modulePath);
  if (argvReal !== null && moduleReal !== null && argvReal === moduleReal) return true;
  return matchesArgvPath(modulePath, argvPath);
}

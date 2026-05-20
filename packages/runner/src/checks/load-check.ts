import { existsSync } from 'node:fs';
import { createRequire } from 'node:module';
import { dirname, join, resolve } from 'node:path';
import { pathToFileURL } from 'node:url';
import { loadConfig } from '@mayjournal/fitness-shared';
import type { Check } from '../types/index.types.js';
import { enUS } from '../runner/enUS.js';

const BUNDLE_DEFAULT_CHECKS = '@mayjournal/fitness-checks/defaultChecks';

const pathLoadedChecks = new WeakSet<Check>();

/** Marks a check loaded from a local path (runs in-process, not via worker). */
export function markPathLoadedCheck(check: Check): void {
  pathLoadedChecks.add(check);
}

/** True when the check was loaded from a path spec, not an npm package. */
export function isPathLoadedCheck(check: Check): boolean {
  return pathLoadedChecks.has(check);
}

/** npm package name for a built-in check. */
export function checkPackageName(name: string): string {
  return `@mayjournal/fitness-check-${name}`;
}

/** True when dir has package.json and node_modules that resolve the checks bundle. */
function dirResolvesChecksBundle(dir: string): boolean {
  const pkgJson = join(dir, 'package.json');
  if (!existsSync(pkgJson) || !existsSync(join(dir, 'node_modules'))) return false;
  try {
    createRequire(pkgJson).resolve(BUNDLE_DEFAULT_CHECKS);
    return true;
  } catch {
    return false;
  }
}

/** Walks up from startRoot to find a directory whose node_modules can resolve fitness checks. */
export function findInstallRoot(startRoot: string): string {
  let dir = resolve(startRoot);
  for (;;) {
    if (dirResolvesChecksBundle(dir)) return dir;
    const parent = dirname(dir);
    if (parent === dir) return startRoot;
    dir = parent;
  }
}

/** Creates a require function rooted at the nearest install directory for check packages. */
function createRequireForRoot(root: string): NodeRequire {
  const installRoot = findInstallRoot(root);
  return createRequire(join(installRoot, 'package.json'));
}

/** Dedupes check names while preserving first occurrence order. */
function dedupeCheckNames(names: readonly string[]): string[] {
  const seen = new Set<string>();
  return names.filter((name) => !seen.has(name) && (seen.add(name), true));
}

/** Reads default check names from the installed checks bundle. */
async function readBundleDefaultCheckNames(root: string): Promise<string[]> {
  const resolved = createRequireForRoot(root).resolve(BUNDLE_DEFAULT_CHECKS);
  const mod = (await import(pathToFileURL(resolved).href)) as {
    default?: readonly string[];
    defaultChecks?: readonly string[];
  };
  const list = mod.defaultChecks ?? mod.default;
  if (!Array.isArray(list) || list.length === 0) throw new Error('empty defaultChecks');
  return [...list];
}

/** Resolves ordered check names: `.fitnessrc` `checks`, else bundle `defaultChecks`. */
export async function resolveCheckNames(root: string): Promise<string[]> {
  const config = loadConfig(root);
  if (config?.checks?.length) return dedupeCheckNames(config.checks);
  try {
    return await readBundleDefaultCheckNames(root);
  } catch {
    throw new Error(enUS.NoChecksConfigured);
  }
}

/** Validates a loaded module default export is a Check with the expected name. */
function assertCheckExport(check: unknown, name: string, pkg: string): Check {
  if (
    check == null ||
    typeof check !== 'object' ||
    typeof (check as Check).run !== 'function' ||
    (check as Check).name !== name
  ) {
    throw new Error(enUS.InvalidCheckExport.replace('{{pkg}}', pkg).replace('{{name}}', name));
  }
  return check as Check;
}

/** Loads a check package; throws when missing or `default.name` does not match. */
export async function loadCheck(name: string, root: string): Promise<Check> {
  const pkg = checkPackageName(name);
  let resolved: string;
  try {
    resolved = createRequireForRoot(root).resolve(pkg);
  } catch {
    throw new Error(enUS.CheckPackageNotInstalled.replace('{{pkg}}', pkg));
  }
  const mod = (await import(pathToFileURL(resolved).href)) as { default?: unknown };
  return assertCheckExport(mod.default, name, pkg);
}

/** Loads a check by name; returns null when the package is missing or invalid. */
export async function tryLoadCheck(name: string, root: string): Promise<Check | null> {
  try {
    return await loadCheck(name, root);
  } catch {
    return null;
  }
}

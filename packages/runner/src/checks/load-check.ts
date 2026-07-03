import { existsSync } from 'node:fs';
import { createRequire } from 'node:module';
import { dirname, join, resolve } from 'node:path';
import { pathToFileURL } from 'node:url';
import { loadConfig } from '@mayjournal/fitness-shared';
import type { Check } from '../types/index.types.js';
import { enUS } from '../runner/enUS.js';

/** True if spec looks like a file path (for loading a check module) rather than an npm check name. */
export function isPathSpec(spec: string): boolean {
  return /[/\\]/.test(spec) || /\.(?:js|mjs|cjs|ts)$/i.test(spec);
}

/** True if value looks like a Check. */
function isCheckLike(v: unknown): v is Check {
  return !!v && typeof (v as Check).run === 'function' && (v as Check).name != null;
}

/** Returns default or first Check-like export from module. */
function getCheckFromModule(mod: { [k: string]: unknown; default?: unknown }): Check | null {
  if (isCheckLike(mod.default)) return mod.default;
  for (const v of Object.values(mod)) if (isCheckLike(v)) return v;
  return null;
}

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

/** npm subpath for a bundled check in @mayjournal/fitness-checks. */
export function checkPackageName(name: string): string {
  return `@mayjournal/fitness-checks/checks/${name}`;
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

/** Loads a Check from a module path (default or first Check-like export); null when missing or invalid. */
export async function loadCheckFromPath(root: string, spec: string): Promise<Check | null> {
  const abs = resolve(root, spec);
  if (!existsSync(abs)) return null;
  try {
    const url = pathToFileURL(abs).href;
    const mod = (await import(url)) as { [k: string]: unknown; default?: unknown };
    const check = getCheckFromModule(mod);
    if (check) markPathLoadedCheck(check);
    return check;
  } catch {
    return null;
  }
}

/** Loads a path spec check; throws when missing or invalid — path specs are explicitly configured. */
export async function loadCheckFromPathOrThrow(root: string, spec: string): Promise<Check> {
  const check = await loadCheckFromPath(root, spec);
  if (!check) throw new Error(enUS.CheckPathInvalid.replace('{{path}}', spec));
  return check;
}

/** Creates a require function rooted at the nearest install directory for check packages. */
function createRequireForRoot(root: string): NodeRequire {
  const installRoot = findInstallRoot(root);
  return createRequire(join(installRoot, 'package.json'));
}

/** Dedupes an ordered spec list; name specs by value, path specs by resolved absolute path. */
function dedupeCheckSpecs(specs: readonly string[], root: string): string[] {
  const seenNames = new Set<string>();
  const seenPaths = new Set<string>();
  return specs.filter((spec) => {
    if (isPathSpec(spec)) {
      const abs = resolve(root, spec);
      if (seenPaths.has(abs)) return false;
      seenPaths.add(abs);
      return true;
    }
    if (seenNames.has(spec)) return false;
    seenNames.add(spec);
    return true;
  });
}

/** Removes name specs listed in `disabledChecks`; path specs are opt-in only and never removed. */
function applyDisabledChecks(specs: string[], disabled?: readonly string[]): string[] {
  if (!disabled?.length) return specs;
  const disabledSet = new Set(disabled);
  return specs.filter((spec) => isPathSpec(spec) || !disabledSet.has(spec));
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

/** Resolves base check specs (names and/or paths) from `.fitnessrc` `checks` or bundle `defaultChecks`. */
async function resolveBaseCheckNames(
  root: string,
  config: ReturnType<typeof loadConfig>
): Promise<string[]> {
  if (config?.checks?.length) return dedupeCheckSpecs(config.checks, root);
  try {
    return await readBundleDefaultCheckNames(root);
  } catch {
    throw new Error(enUS.NoChecksConfigured);
  }
}

/** Resolves ordered check specs (names and/or paths): `.fitnessrc` `checks`, else bundle `defaultChecks`, minus `disabledChecks` (name specs only). */
export async function resolveCheckNames(root: string): Promise<string[]> {
  const config = loadConfig(root);
  const specs = await resolveBaseCheckNames(root, config);
  return applyDisabledChecks(specs, config?.disabledChecks);
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

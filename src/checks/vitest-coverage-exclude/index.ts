import { existsSync, readFileSync } from 'node:fs';
import { createRequire } from 'node:module';
import { join } from 'node:path';
import { CheckName } from '../../types/index.types.js';
import type { Check } from '../../types/index.types.js';

const require = createRequire(import.meta.url);

/** Allowed coverage exclude patterns: declaration and type-only; Vitest excludes tests by default. */
export const ALLOWED_COVERAGE_EXCLUDE_PATTERNS = [
  '**/*.d.ts',
  '**/*.types.ts',
  '**/*.test.ts',
  '**/*.spec.ts',
] as const;

export const VITEST_CONFIG_NAMES = [
  'vitest.config.ts',
  'vitest.config.js',
  'vitest.config.mts',
  'vitest.config.mjs',
];

/** Allowed coverage exclude suffixes: declaration, type-only, and test files. */
export const ALLOWED_SUFFIXES = /\.(d\.ts|types\.ts|test\.ts|spec\.ts)$/;

/** Returns true if the exclude pattern is not in the allowed conventional set (.d.ts, .types.ts, .test.ts, .spec.ts). */
function isDisallowedTsPattern(pattern: string): boolean {
  const t = pattern.trim();
  if (t.endsWith('/**')) return true;
  if (!t.includes('.ts')) return false;
  if (ALLOWED_SUFFIXES.test(t)) return false;
  return true;
}

type VitestConfig = {
  test?: { coverage?: { exclude?: string[] } };
  coverage?: { exclude?: string[] };
};

/** True if x is a non-null object. */
function isObject(x: unknown): x is object {
  return typeof x === 'object' && x != null;
}

/** Extract Vitest config from loaded module (default export or module itself). Exported for tests. */
export function configFromMod(mod: unknown): VitestConfig | null {
  if (!isObject(mod)) return null;
  const def = (mod as { default?: unknown }).default;
  if (isObject(def)) return def as VitestConfig;
  return mod as VitestConfig;
}

/** Try to load a single vitest.config.* file; returns null if missing or invalid. */
function tryLoadConfigFile(root: string, name: string): VitestConfig | null {
  const path = join(root, name);
  if (!existsSync(path)) return null;
  try {
    const jiti = require('jiti')(root, { esmResolve: true });
    return configFromMod(jiti(path));
  } catch {
    return null;
  }
}

/** Extract vitest config from parsed package.json. */
function getVitestFromPkg(pkg: { vitest?: VitestConfig }): VitestConfig | null {
  const vitest = pkg.vitest;
  return vitest != null && typeof vitest === 'object' ? vitest : null;
}

/** Try to load vitest config from package.json "vitest" key. */
function tryLoadPackageJsonVitest(root: string): VitestConfig | null {
  const pkgPath = join(root, 'package.json');
  if (!existsSync(pkgPath)) return null;
  try {
    const pkg = JSON.parse(readFileSync(pkgPath, 'utf8')) as { vitest?: VitestConfig };
    return getVitestFromPkg(pkg);
  } catch {
    return null;
  }
}

/** Load Vitest config object from root (vitest.config.* or package.json vitest key). */
function loadVitestConfig(root: string): VitestConfig | null {
  for (const name of VITEST_CONFIG_NAMES) {
    const config = tryLoadConfigFile(root, name);
    if (config != null) return config;
  }
  return tryLoadPackageJsonVitest(root);
}

/** Get exclude array from config (test.coverage.exclude or coverage.exclude). */
function getExcludeFromConfig(config: VitestConfig): string[] | undefined {
  const testExclude = config.test?.coverage?.exclude;
  if (testExclude != null) return testExclude;
  return config.coverage?.exclude;
}

/** Get coverage exclude array from Vitest config (test.coverage.exclude or coverage.exclude). */
function getCoverageExclude(config: VitestConfig | null): string[] {
  if (config == null) return [];
  const exclude = getExcludeFromConfig(config);
  if (!Array.isArray(exclude)) return [];
  return exclude;
}

/** Ensures Vitest coverage exclude only uses conventional patterns (e.g. *.d.ts, *.types.ts); Vitest excludes tests by default. */
export const vitestCoverageExcludeCheck: Check = {
  name: CheckName.VitestCoverageExclude,
  async run(root = process.cwd()) {
    const config = loadVitestConfig(root);
    const exclude = getCoverageExclude(config);
    if (exclude.length === 0) return { errors: [], meta: { filesChecked: 1 }, ok: true };
    const bad = exclude.filter((p) => typeof p === 'string' && isDisallowedTsPattern(p));
    if (bad.length === 0) return { errors: [], meta: { filesChecked: 1 }, ok: true };
    const allowed = ALLOWED_COVERAGE_EXCLUDE_PATTERNS.join(', ');
    const errors = bad.map(
      (p) => `Vitest coverage exclude only allows ${allowed}; disallowed: ${JSON.stringify(p)}`
    );
    return { errors, meta: { filesChecked: 1 }, ok: false };
  },
};

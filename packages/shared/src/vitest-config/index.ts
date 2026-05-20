import { existsSync, readFileSync } from 'node:fs';
import { createRequire } from 'node:module';
import { join } from 'node:path';

const require = createRequire(import.meta.url);

export const VITEST_CONFIG_NAMES = [
  'vitest.config.cjs',
  'vitest.config.js',
  'vitest.config.mjs',
  'vitest.config.mts',
  'vitest.config.ts',
];

type CoverageBlock = {
  exclude?: string[];
  thresholds?: {
    branches?: number;
    functions?: number;
    lines?: number;
    statements?: number;
  };
};

/** Minimal Vitest config shape for coverage exclude and thresholds. */
export type VitestConfigRaw = {
  coverage?: CoverageBlock;
  test?: { coverage?: CoverageBlock };
};

/** True if x is a non-null object. */
export function isObject(x: unknown): x is object {
  return typeof x === 'object' && x != null;
}

/** Extract config from loaded module (default export or module itself). */
export function configFromMod(mod: unknown): VitestConfigRaw | null {
  if (!isObject(mod)) return null;
  const def = (mod as { default?: unknown }).default;
  if (isObject(def)) return def as VitestConfigRaw;
  return mod as VitestConfigRaw;
}

/** Try to load a single vitest.config.* file (require for .cjs/.js, jiti otherwise). */
export function tryLoadConfigFile(root: string, name: string): VitestConfigRaw | null {
  const path = join(root, name);
  if (!existsSync(path)) return null;
  try {
    if (name.endsWith('.cjs') || name.endsWith('.js')) {
      return configFromMod(require(path));
    }
    const jiti = require('jiti')(root, { esmResolve: true });
    return configFromMod(jiti(path));
  } catch {
    return null;
  }
}

/** Try to load vitest config from package.json "vitest" key. */
export function tryLoadPackageJsonVitest(root: string): VitestConfigRaw | null {
  const pkgPath = join(root, 'package.json');
  if (!existsSync(pkgPath)) return null;
  try {
    const pkg = JSON.parse(readFileSync(pkgPath, 'utf8')) as { vitest?: VitestConfigRaw };
    return pkg.vitest != null && typeof pkg.vitest === 'object' ? pkg.vitest : null;
  } catch {
    return null;
  }
}

/** Load Vitest config from a single directory (config files then package.json vitest key). */
export function loadVitestConfigFromRoot(root: string): VitestConfigRaw | null {
  for (const name of VITEST_CONFIG_NAMES) {
    const config = tryLoadConfigFile(root, name);
    if (config != null) return config;
  }
  return tryLoadPackageJsonVitest(root);
}

/** Load Vitest config from root; falls back to fallbackRoot (e.g. @mayjournal/fitness package). */
export function loadVitestConfig(root: string, fallbackRoot?: string): VitestConfigRaw | null {
  const local = loadVitestConfigFromRoot(root);
  if (local != null) return local;
  if (fallbackRoot == null) return null;
  return loadVitestConfigFromRoot(fallbackRoot);
}

/** Get coverage block from config (test.coverage ?? coverage). */
export function getCoverageBlock(config: VitestConfigRaw | null): CoverageBlock | undefined {
  if (!config) return undefined;
  return config.test?.coverage ?? config.coverage;
}

/** Get exclude array from config (test.coverage.exclude ?? coverage.exclude). */
export function getCoverageExcludeFromConfig(config: VitestConfigRaw | null): string[] | undefined {
  const block = getCoverageBlock(config);
  return block?.exclude;
}

/** Get thresholds from config. */
export function getThresholdsFromConfig(
  config: VitestConfigRaw | null
): CoverageBlock['thresholds'] | undefined {
  return getCoverageBlock(config)?.thresholds;
}

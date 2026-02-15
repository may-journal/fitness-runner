import { existsSync } from 'node:fs';
import { createRequire } from 'node:module';
import { join } from 'node:path';
import type { FitnessConfig } from '../types/index.js';

const require = createRequire(import.meta.url);

/**
 *
 */
/** True if raw has no default or default is non-null. */
function hasValidDefault(raw: object): boolean {
  if (!('default' in raw)) return true;
  return (raw as { default?: unknown }).default != null;
}

/** Resolves default export or the module itself. */
function getRawExport(mod: unknown): unknown {
  if (mod == null || typeof mod !== 'object') return null;
  if ('default' in (mod as object) && (mod as { default?: unknown }).default !== undefined) {
    return (mod as { default: unknown }).default;
  }
  return mod;
}

/** Normalizes required/exported config from a loaded module. */
function parseConfigModule(mod: unknown): FitnessConfig | null {
  const raw = getRawExport(mod);
  if (raw == null || typeof raw !== 'object') return null;
  if (!hasValidDefault(raw as object)) return null;
  return raw as FitnessConfig;
}

/** Loads one config file by name; returns null if missing or invalid. */
function loadOneConfig(root: string, name: string): FitnessConfig | null {
  const path = join(root, name);
  if (!existsSync(path)) return null;
  try {
    const jiti = require('jiti')(root, { esmResolve: true });
    const mod = jiti(path);
    return parseConfigModule(mod);
  } catch {
    return null;
  }
}

/** Loads .fitnessrc.ts or .fitnessrc.js from root; returns null if missing or invalid. */
export function loadConfig(root: string): FitnessConfig | null {
  for (const name of ['.fitnessrc.ts',
    '.fitnessrc.js']) {
    const config = loadOneConfig(root, name);
    if (config != null) return config;
  }
  return null;
}

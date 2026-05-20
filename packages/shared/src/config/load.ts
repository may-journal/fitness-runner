import { existsSync } from 'node:fs';
import { createRequire } from 'node:module';
import { join } from 'node:path';
import type { FitnessConfig } from '../types/fitness-config.types.js';

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
  const def = (mod as { default?: unknown }).default;
  const useDefault = 'default' in (mod as object) && def !== undefined;
  return useDefault ? def : mod;
}

/** Normalizes required/exported config from a loaded module. */
function parseConfigModule(mod: unknown): FitnessConfig | null {
  const raw = getRawExport(mod);
  if (raw == null || typeof raw !== 'object') return null;
  const valid = hasValidDefault(raw as object);
  return valid ? (raw as FitnessConfig) : null;
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
  for (const name of ['.fitnessrc.ts', '.fitnessrc.js']) {
    const config = loadOneConfig(root, name);
    if (config != null) return config;
  }
  return null;
}

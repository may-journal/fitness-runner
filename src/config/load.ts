import { existsSync } from 'node:fs';
import { createRequire } from 'node:module';
import { join } from 'node:path';
import type { FitnessConfig } from '../types/index.js';

const require = createRequire(import.meta.url);

/** Loads .fitnessrc.ts or .fitnessrc.js from root; returns null if missing or invalid. */
export function loadConfig(root: string): FitnessConfig | null {
  for (const name of ['.fitnessrc.ts', '.fitnessrc.js']) {
    const path = join(root, name);
    if (!existsSync(path)) continue;
    try {
      const jiti = require('jiti')(root, { esmResolve: true });
      const mod = jiti(path);
      const raw = mod != null && 'default' in mod && mod.default !== undefined ? mod.default : mod;
      if (raw == null || typeof raw !== 'object') return null;
      if ('default' in (raw as object) && (raw as { default?: unknown }).default == null) return null;
      return raw as FitnessConfig;
    } catch {
      return null;
    }
  }
  return null;
}

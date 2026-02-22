import { existsSync, readFileSync } from 'node:fs';
import { join } from 'node:path';
import { CheckName } from '../../types/index.types.js';
import type { Check } from '../../types/index.types.js';

const NVMRC = '.nvmrc';
const ERROR_MISSING = 'missing .nvmrc';
const ERROR_VERSION = (current: string, required: number): string =>
  `Node ${current} does not satisfy .nvmrc (requires ${required}.x). Run: nvm use`;

/** Parses major version from a version string (e.g. v24.0.0 or 24). */
function versionMajor(version: string): number {
  return parseInt(version.replace(/^v?(\d+).*$/, '$1'), 10);
}

/** Validates current Node version satisfies .nvmrc at repo root. */
export const nodeVersionCheck: Check = {
  name: CheckName.NodeVersion,
  async run(root = process.cwd()) {
    const path = join(root, NVMRC);
    if (!existsSync(path)) {
      return { errors: [ERROR_MISSING], meta: { filesChecked: 1 }, ok: false };
    }
    const raw = readFileSync(path, 'utf8').trim();
    const requiredMajor = versionMajor(raw);
    const currentMajor = versionMajor(process.version);
    if (currentMajor >= requiredMajor) {
      return { errors: [], meta: { filesChecked: 1 }, ok: true };
    }
    return {
      errors: [ERROR_VERSION(process.version, requiredMajor)],
      meta: { filesChecked: 1 },
      ok: false,
    };
  },
};

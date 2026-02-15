import { existsSync, readFileSync } from 'node:fs';
import { join } from 'node:path';
import type { Check } from '../../types/index.js';

const NVMRC = '.nvmrc';
const ERROR_MISSING = 'missing .nvmrc';
const ERROR_VERSION = (current: string, required: number): string =>
  `Node ${current} does not satisfy .nvmrc (requires ${required}.x). Run: nvm use`;

/** Validates current Node version satisfies .nvmrc at repo root. */
export const nodeVersionCheck: Check = {
  name: 'node-version',
  async run(root = process.cwd()) {
    const path = join(root, NVMRC);
    if (!existsSync(path)) {
      return { ok: false, errors: [ERROR_MISSING], meta: { filesChecked: 1 } };
    }
    const raw = readFileSync(path, 'utf8').trim();
    const requiredMajor = parseInt(raw.replace(/^v?(\d+).*$/, '$1'), 10);
    const currentMajor = parseInt(process.version.replace(/^v?(\d+).*$/, '$1'), 10);
    if (currentMajor >= requiredMajor) {
      return { ok: true, errors: [], meta: { filesChecked: 1 } };
    }
    return { ok: false, errors: [ERROR_VERSION(process.version, requiredMajor)], meta: { filesChecked: 1 } };
  },
};

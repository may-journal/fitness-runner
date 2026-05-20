import { existsSync, readFileSync } from 'node:fs';
import { join } from 'node:path';
import { checkResult } from '@mayjournal/fitness-shared';
import type { Check, CheckName } from '@mayjournal/fitness';

const NODE_VERSION = 'node-version' as CheckName;

const NVMRC = '.nvmrc';
const ERROR_MISSING = 'missing .nvmrc';
const ERROR_VERSION = (current: string, required: number): string =>
  `Node ${current} does not satisfy .nvmrc (requires ${required}.x). Run: nvm use`;

/** Parses major version from a version string (e.g. v24.0.0 or 24). */
function versionMajor(version: string): number {
  return parseInt(version.replace(/^v?(\d+).*$/, '$1'), 10);
}

/** Validates current Node version satisfies .nvmrc at repo root. */
const nodeVersionCheck: Check = {
  name: NODE_VERSION,
  async run(root = process.cwd()) {
    const path = join(root, NVMRC);
    if (!existsSync(path)) return checkResult(false, [ERROR_MISSING], 1);
    const raw = readFileSync(path, 'utf8').trim();
    const requiredMajor = versionMajor(raw);
    const currentMajor = versionMajor(process.version);
    if (currentMajor >= requiredMajor) return checkResult(true, [], 1);
    return checkResult(false, [ERROR_VERSION(process.version, requiredMajor)], 1);
  },
};

export default nodeVersionCheck;

import { existsSync, readFileSync } from 'node:fs';
import { join } from 'node:path';
import type { Check } from '../types/index.js';

const ROOT_CHANGELOG = 'CHANGELOG.md';
const ERROR_MISSING = 'missing root CHANGELOG.md';
const ERROR_INVALID = 'CHANGELOG.md must have at least one dated section (## or ### yyyy-mm-dd)';
const DATED_SECTION_RE = /^#{2,3}\s+\d{4}-\d{2}-\d{2}/m;

/** Returns true if content has at least one ##/### yyyy-mm-dd heading. */
function hasDatedSection(content: string): boolean {
  return DATED_SECTION_RE.test(content);
}

/** Validates repo root has CHANGELOG.md with at least one yyyy-mm-dd section. */
export const changelogCheck: Check = {
  name: 'changelog',
  async run(root = process.cwd()) {
    const path = join(root, ROOT_CHANGELOG);
    if (!existsSync(path)) {
      return { ok: false, errors: [ERROR_MISSING], meta: { filesChecked: 1 } };
    }
    const content = readFileSync(path, 'utf8');
    if (!hasDatedSection(content)) {
      return { ok: false, errors: [ERROR_INVALID], meta: { filesChecked: 1 } };
    }
    return { ok: true, errors: [], meta: { filesChecked: 1 } };
  },
};

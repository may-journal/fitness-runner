import { existsSync, readFileSync } from 'node:fs';
import { join } from 'node:path';
import type { Check } from '../../types/index.js';

const ROOT_CHANGELOG = 'CHANGELOG.md';
const ERROR_MISSING = 'missing root CHANGELOG.md';
const DATED_SECTION_RE = /^###\s+\d{4}-\d{2}-\d{2}@\S+/;

/** Returns lines that are ### headings (level-3 only). */
function getH3Lines(content: string): string[] {
  return content.split('\n').filter((line) => /^###\s/.test(line));
}

/** Validates repo root has CHANGELOG.md; every ### heading must be ### yyyy-mm-dd@time. */
export const changelogCheck: Check = {
  name: 'changelog',
  async run(root = process.cwd()) {
    const path = join(root, ROOT_CHANGELOG);
    if (!existsSync(path)) {
      return { ok: false, errors: [ERROR_MISSING], meta: { filesChecked: 1 } };
    }
    const content = readFileSync(path, 'utf8');
    const h3s = getH3Lines(content);
    if (h3s.length === 0) {
      return { ok: false, errors: ['CHANGELOG.md must have at least one ### yyyy-mm-dd@time section'], meta: { filesChecked: 1 } };
    }
    const invalid = h3s.filter((line) => !DATED_SECTION_RE.test(line));
    if (invalid.length > 0) {
      return {
        ok: false,
        errors: invalid.map((line) => `every ### heading must be ### yyyy-mm-dd@time (invalid: "${line.trim()}")`),
        meta: { filesChecked: 1 },
      };
    }
    return { ok: true, errors: [], meta: { filesChecked: 1 } };
  },
};

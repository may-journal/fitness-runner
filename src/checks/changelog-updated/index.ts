import { existsSync, readFileSync } from 'node:fs';
import { execSync } from 'node:child_process';
import { join } from 'node:path';
import type { Check } from '../../types/index.js';

const ROOT_CHANGELOG = 'CHANGELOG.md';
const MIN_OVERLAP = 3;
const MIN_WORD_LEN = 3;

function extractWords(text: string): Set<string> {
  const words = new Set<string>();
  const lower = text.toLowerCase().replace(/[^a-z0-9\s]/g, ' ');
  for (const w of lower.split(/\s+/)) {
    if (w.length >= MIN_WORD_LEN) words.add(w);
  }
  return words;
}

function getStagedDiffWords(root: string): Set<string> {
  const out = execSync('git diff --cached', { encoding: 'utf8', cwd: root });
  const lines = out.split('\n').filter((l) => l.startsWith('+') && !l.startsWith('+++'));
  const text = lines.map((l) => l.slice(1)).join(' ');
  return extractWords(text);
}

/** When --staged: ensures CHANGELOG.md contains at least MIN_OVERLAP words also present in staged diff. */
export const changelogUpdatedCheck: Check = {
  name: 'changelog-updated',
  async run(root = process.cwd(), context) {
    const staged = context?.stagedFiles;
    if (!staged?.length) return { ok: true, errors: [], meta: { filesChecked: 0 } };
    const path = join(root, ROOT_CHANGELOG);
    if (!existsSync(path)) {
      return {
        ok: false,
        errors: ['CHANGELOG.md missing; add it and mention your staged changes'],
        meta: { filesChecked: 1 },
      };
    }
    const diffWords = getStagedDiffWords(root);
    if (diffWords.size === 0) return { ok: true, errors: [], meta: { filesChecked: 1 } };
    const changelogContent = readFileSync(path, 'utf8');
    const changelogWords = extractWords(changelogContent);
    const overlap = [...diffWords].filter((w) => changelogWords.has(w));
    if (overlap.length >= MIN_OVERLAP) {
      return { ok: true, errors: [], meta: { filesChecked: 1 } };
    }
    return {
      ok: false,
      errors: [
        `CHANGELOG.md should mention at least ${MIN_OVERLAP} words from your staged changes (found ${overlap.length}: ${overlap.slice(0, 5).join(', ')})`,
      ],
      meta: { filesChecked: 1 },
    };
  },
};

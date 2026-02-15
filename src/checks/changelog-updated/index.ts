import { existsSync } from 'node:fs';
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

/** Returns map of file path (repo-relative) -> added line content. */
function getStagedDiffByFile(root: string): Map<string, string> {
  const out = execSync('git diff --cached', { encoding: 'utf8', cwd: root });
  const byFile = new Map<string, string>();
  let current = '';
  for (const line of out.split('\n')) {
    if (line.startsWith('+++ ')) {
      current = line.slice(4).trim().replace(/^b\//, '');
    } else if (line.startsWith('+') && !line.startsWith('++') && current) {
      const prev = byFile.get(current) ?? '';
      byFile.set(current, prev ? prev + ' ' + line.slice(1) : line.slice(1));
    }
  }
  return byFile;
}

/** When --staged: ensures added lines in CHANGELOG.md share MIN_OVERLAP words with rest of staged diff. */
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
    const byFile = getStagedDiffByFile(root);
    const changelogAdded = byFile.get(ROOT_CHANGELOG) ?? '';
    const changelogWords = extractWords(changelogAdded);
    if (changelogWords.size === 0) {
      return {
        ok: false,
        errors: ['Stage CHANGELOG.md and add an entry that mentions your staged changes'],
        meta: { filesChecked: 1 },
      };
    }
    const restLines: string[] = [];
    for (const [file, content] of byFile) {
      if (file !== ROOT_CHANGELOG) restLines.push(content);
    }
    const restWords = extractWords(restLines.join(' '));
    if (restWords.size === 0) return { ok: true, errors: [], meta: { filesChecked: 1 } };
    const overlap = [...changelogWords].filter((w) => restWords.has(w));
    if (overlap.length >= MIN_OVERLAP) {
      return { ok: true, errors: [], meta: { filesChecked: 1 } };
    }
    return {
      ok: false,
      errors: [
        `CHANGELOG.md additions should mention at least ${MIN_OVERLAP} words from your staged changes (found ${overlap.length}: ${overlap.slice(0, 5).join(', ')})`,
      ],
      meta: { filesChecked: 1 },
    };
  },
};

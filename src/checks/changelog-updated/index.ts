import { existsSync } from 'node:fs';
import { execSync } from 'node:child_process';
import { join } from 'node:path';
import type { Check } from '../../types/index.types.js';

const ROOT_CHANGELOG = 'CHANGELOG.md';
const MIN_OVERLAP = 3;
const MIN_WORD_LEN = 3;
const SUGGEST_WORDS = 10;

/** Extract words of at least MIN_WORD_LEN from text (lowercased, alphanumeric). */
function extractWords(text: string): Set<string> {
  const words = new Set<string>();
  const lower = text.toLowerCase().replace(/[^a-z0-9\s]/g, ' ');
  for (const w of lower.split(/\s+/)) {
    if (w.length >= MIN_WORD_LEN) words.add(w);
  }
  return words;
}

/** Returns up to n random words from the set (for suggestion in errors). */
function sampleWords(words: Set<string>, n: number): string[] {
  const arr = [...words];
  for (let i = arr.length - 1; i > 0 && n > 0; i--) {
    const j = Math.floor(Math.random() * (i + 1));
    [arr[i],
      arr[j]] = [arr[j],
      arr[i]];
  }
  return arr.slice(0, n);
}

/** Parses one git diff line; returns updated current file and optional added line. */
function processDiffLine(
  line: string,
  current: string,
): { current: string; file?: string; text?: string } {
  if (line.startsWith('+++ ')) return { current: line.slice(4).trim().replace(/^b\//, '') };
  if (line.startsWith('+') && !line.startsWith('++') && current) {
    return { current, file: current, text: line.slice(1) };
  }
  return { current };
}

/**
 *
 */
function appendDiffLine(byFile: Map<string, string>, file: string, text: string): void {
  const prev = byFile.get(file) ?? '';
  byFile.set(file, prev ? prev + ' ' + text : text);
}

/** Returns map of file path (repo-relative) -> added line content. */
function getStagedDiffByFile(root: string): Map<string, string> {
  const out = execSync('git diff --cached', { encoding: 'utf8', cwd: root });
  const byFile = new Map<string, string>();
  let current = '';
  for (const line of out.split('\n')) {
    const next = processDiffLine(line, current);
    current = next.current;
    if (next.file != null && next.text != null) appendDiffLine(byFile, next.file, next.text);
  }
  return byFile;
}

/** Words from staged diff excluding CHANGELOG.md. */
function getRestWordsFromDiff(byFile: Map<string, string>): Set<string> {
  const restLines: string[] = [];
  for (const [file,
    content] of byFile) {
    if (file !== ROOT_CHANGELOG) restLines.push(content);
  }
  return extractWords(restLines.join(' '));
}

/** Checks overlap count and returns pass or error result with suggestion. */
function checkOverlapAndReport(
  changelogWords: Set<string>,
  restWords: Set<string>,
): { ok: boolean; errors: string[]; meta: { filesChecked: number } } {
  const overlap = [...changelogWords].filter((w) => restWords.has(w));
  if (overlap.length >= MIN_OVERLAP) return { ok: true, errors: [], meta: { filesChecked: 1 } };
  const suggested = sampleWords(restWords, SUGGEST_WORDS);
  const errors = [
    `CHANGELOG.md additions should mention at least ${MIN_OVERLAP} words from your staged changes (found ${overlap.length}: ${overlap.slice(0, 5).join(', ')})`,
    `e.g. use words like: ${suggested.join(', ')}`,
  ];
  return { ok: false, errors, meta: { filesChecked: 1 } };
}

/** Returns early result if no staged files or CHANGELOG missing; null to continue. */
function ensureChangelogExists(
  root: string,
  context: { stagedFiles?: string[] } | undefined,
): { ok: true; errors: []; meta: { filesChecked: number } } | { ok: false; errors: string[]; meta: { filesChecked: number } } | null {
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
  return null;
}

type ChangelogEarlyResult = { ok: boolean; errors: string[]; meta: { filesChecked: number } };
type ChangelogOverlapInput =
  | { err: ChangelogEarlyResult }
  | { changelogWords: Set<string>; restWords: Set<string> };

/** Builds changelog/rest word sets or early result for overlap check. */
function getChangelogOverlapInput(root: string): ChangelogOverlapInput {
  const byFile = getStagedDiffByFile(root);
  const changelogWords = extractWords(byFile.get(ROOT_CHANGELOG) ?? '');
  if (changelogWords.size === 0) {
    return {
      err: {
        ok: false,
        errors: ['Stage CHANGELOG.md and add an entry that mentions your staged changes'],
        meta: { filesChecked: 1 },
      },
    };
  }
  const restWords = getRestWordsFromDiff(byFile);
  if (restWords.size === 0) return { err: { ok: true, errors: [], meta: { filesChecked: 1 } } };
  return { changelogWords, restWords };
}

/** When context has stagedFiles, ensures CHANGELOG.md additions share MIN_OVERLAP words with rest of staged diff. */
export const changelogUpdatedCheck: Check = {
  name: 'changelog-updated',
  async run(root = process.cwd(), context) {
    const early = ensureChangelogExists(root, context);
    if (early != null) return early;
    const next = getChangelogOverlapInput(root);
    if ('err' in next) return next.err;
    return checkOverlapAndReport(next.changelogWords, next.restWords);
  },
};

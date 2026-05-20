import { existsSync, readFileSync } from 'node:fs';
import { execSync } from 'node:child_process';
import { join } from 'node:path';
import type { Check, CheckName } from '@mayjournal/fitness';
import {
  checkResult,
  execSyncResult,
  getExecSync,
  getStagedFiles,
} from '@mayjournal/fitness-shared';
import type { ExecSyncFn, RunContext } from '@mayjournal/fitness-shared';

const ROOT_CHANGELOG = 'CHANGELOG.md';
const MIN_OVERLAP = 3;
const MIN_WORD_LEN = 3;
const SUGGEST_WORDS = 10;

const HEADING_RE = /### (\d{4}\.\d{2}\.\d{2}\.\d{4})/g;

export const MSG_CHANGELOG_MISSING = 'CHANGELOG.md missing; add it and mention your staged changes';
export const MSG_STAGE_CHANGELOG =
  'Stage CHANGELOG.md and add an entry that mentions your staged changes';
export const MSG_OVERLAP_HEAD = 'CHANGELOG.md additions should mention at least ';
export const MSG_OVERLAP_TAIL = ' words from your staged changes (found ';
export const MSG_SUGGEST_PREFIX = 'e.g. use words like: ';
export const MSG_CHANGELOG_TIME =
  'CHANGELOG.md new section heading must use current date and time (yyyy.mm.dd.HHMM), not a guessed time';

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
    [arr[i], arr[j]] = [arr[j], arr[i]];
  }
  return arr.slice(0, n);
}

/** Parses one git diff line; returns updated current file and optional added line. */
function processDiffLine(
  line: string,
  current: string
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
function getStagedDiffByFile(root: string, execFn: ExecSyncFn = execSync): Map<string, string> {
  const { output: out } = execSyncResult(root, 'git diff --cached', execFn);
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
  for (const [file, content] of byFile) {
    if (file !== ROOT_CHANGELOG) restLines.push(content);
  }
  return extractWords(restLines.join(' '));
}

const VERSION_TS_RE = /(\d{4}\.\d{2}\.\d{2}\.\d{4})$/;

/** Format date as yyyy.mm.dd.HHMM for changelog heading. */
function getExpectedChangelogTimestamp(now: Date): string {
  const y = now.getFullYear();
  const m = String(now.getMonth() + 1).padStart(2, '0');
  const d = String(now.getDate()).padStart(2, '0');
  const h = String(now.getHours()).padStart(2, '0');
  const min = String(now.getMinutes()).padStart(2, '0');
  return `${y}.${m}.${d}.${h}${min}`;
}

/** Match ensure-changelog-timestamp: heading aligns with root package version suffix. */
function resolveExpectedChangelogTimestamp(root: string, now: Date): string {
  try {
    const pkg = JSON.parse(readFileSync(join(root, 'package.json'), 'utf8')) as {
      version?: string;
    };
    const m = pkg.version?.match(VERSION_TS_RE);
    if (m) return m[1];
  } catch {
    /* fall back to wall clock */
  }
  return getExpectedChangelogTimestamp(now);
}

/** Extract all ### yyyy.mm.dd.HHMM timestamps from added changelog content. */
function extractHeadingTimestamps(addedContent: string): string[] {
  const out: string[] = [];
  let m: RegExpExecArray | null;
  HEADING_RE.lastIndex = 0;
  while ((m = HEADING_RE.exec(addedContent)) !== null) out.push(m[1]);
  return out;
}

/** Require every new changelog heading timestamp to match current date and time. */
function checkChangelogTime(
  root: string,
  addedContent: string,
  now: Date
): { errors: string[]; ok: false } | { ok: true } {
  const timestamps = extractHeadingTimestamps(addedContent);
  if (timestamps.length === 0) return { ok: true };
  const expected = resolveExpectedChangelogTimestamp(root, now);
  const bad = timestamps.filter((t) => t !== expected);
  if (bad.length === 0) return { ok: true };
  return { errors: [MSG_CHANGELOG_TIME + ` (expected ### ${expected})`], ok: false };
}

/** Checks overlap count and returns pass or error result with suggestion. */
function checkOverlapAndReport(changelogWords: Set<string>, restWords: Set<string>) {
  const overlap = [...changelogWords].filter((w) => restWords.has(w));
  if (overlap.length >= MIN_OVERLAP) return checkResult(true, [], 1);
  const suggested = sampleWords(restWords, SUGGEST_WORDS);
  const errors = [
    MSG_OVERLAP_HEAD +
      MIN_OVERLAP +
      MSG_OVERLAP_TAIL +
      overlap.length +
      ': ' +
      overlap.slice(0, 5).join(', ') +
      ')',
    MSG_SUGGEST_PREFIX + suggested.join(', '),
  ];
  return checkResult(false, errors, 1);
}

/** Returns early result if no staged files or CHANGELOG missing; null to continue. */
function ensureChangelogExists(
  root: string,
  context: RunContext | undefined
): ReturnType<typeof checkResult> | null {
  const staged = getStagedFiles(context);
  if (!staged?.length) return checkResult(true, [], 0);
  const path = join(root, ROOT_CHANGELOG);
  if (!existsSync(path)) return checkResult(false, [MSG_CHANGELOG_MISSING], 1);
  return null;
}

type ChangelogOverlapInput =
  | { err: ReturnType<typeof checkResult> }
  | { changelogAddedContent: string; changelogWords: Set<string>; restWords: Set<string> };

/** Builds changelog/rest word sets or early result for overlap check. */
function getChangelogOverlapInput(
  root: string,
  execFn: ExecSyncFn = execSync
): ChangelogOverlapInput {
  const byFile = getStagedDiffByFile(root, execFn);
  const changelogAddedContent = byFile.get(ROOT_CHANGELOG) ?? '';
  const changelogWords = extractWords(changelogAddedContent);
  if (changelogWords.size === 0) return { err: checkResult(false, [MSG_STAGE_CHANGELOG], 1) };
  const restWords = getRestWordsFromDiff(byFile);
  if (restWords.size === 0) return { err: checkResult(true, [], 1) };
  return { changelogAddedContent, changelogWords, restWords };
}

/** Runs time check then overlap; returns report or time error. */
function runTimeAndOverlap(
  root: string,
  next: { changelogAddedContent: string; changelogWords: Set<string>; restWords: Set<string> },
  context: RunContext | undefined
): ReturnType<typeof checkResult> {
  const now = (context?._now ?? (() => new Date()))();
  const timeResult = checkChangelogTime(root, next.changelogAddedContent, now);
  if (!timeResult.ok) return checkResult(false, timeResult.errors, 1);
  return checkOverlapAndReport(next.changelogWords, next.restWords);
}

/** Runs time then overlap check; returns early result or report. */
function runChangelogUpdated(
  root: string,
  context: RunContext | undefined
): ReturnType<typeof checkResult> {
  const early = ensureChangelogExists(root, context);
  if (early != null) return early;
  const next = getChangelogOverlapInput(root, getExecSync(context));
  if ('err' in next) return next.err;
  return runTimeAndOverlap(root, next, context);
}

/** When context has stagedFiles, ensures CHANGELOG.md additions share MIN_OVERLAP words with rest of staged diff. */
export const changelogUpdatedCheck: Check = {
  name: 'changelog-updated' as CheckName,
  async run(root = process.cwd(), context?: RunContext) {
    return runChangelogUpdated(root, context);
  },
};

export default changelogUpdatedCheck;

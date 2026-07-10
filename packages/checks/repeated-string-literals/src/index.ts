import { readFileSync } from 'node:fs';
import { join } from 'node:path';
import { checkResult, findFilesByExtension, loadConfig } from '@mayjournal/fitness-shared';
import type { Check, CheckName } from '@mayjournal/fitness';
import { scanStringLiterals, type Literal } from './scan.js';

export { IDIOMATIC_VALUES, MIN_LENGTH, scanStringLiterals, type Literal } from './scan.js';

/** Source extensions scanned. Test/spec/bench files are excluded — fixtures legitimately repeat strings. */
export const SOURCE_EXTENSIONS = ['.cjs', '.cts', '.js', '.mjs', '.mts', '.ts', '.tsx'];

/** A string literal value must appear at least this many times (repo-wide) to be flagged. */
export const MIN_OCCURRENCES = 3;

/** Cap on locations listed per duplicated value to keep an error line readable. */
export const MAX_LOCATIONS = 5;

/** Matches a test/spec/bench source file (fixtures there repeat strings intentionally). */
const FIXTURE_FILE_RE = /\.(?:bench|spec|test)\.(?:c|m)?[jt]sx?$/;

/** One location of a duplicated value. */
type Location = { file: string; line: number };

/** Returns sorted, de-duplicated relative paths of scanned source files (test/spec/bench excluded). */
export async function findSourceFiles(root: string): Promise<string[]> {
  const groups = await Promise.all(SOURCE_EXTENSIONS.map((ext) => findFilesByExtension(root, ext)));
  return [...new Set(groups.flat())].filter((f) => !FIXTURE_FILE_RE.test(f)).sort();
}

/** Exact values allowed via `.fitnessrc` `repeatedStringLiterals.allow` (project baseline). */
export function getAllowedValues(root: string): Set<string> {
  const allow = loadConfig(root)?.repeatedStringLiterals?.allow ?? [];
  return new Set(allow.filter((v) => typeof v === 'string'));
}

/** Groups scanned literals by value into their locations across all files. */
function groupByValue(scanned: { file: string; literals: Literal[] }[]): Map<string, Location[]> {
  const byValue = new Map<string, Location[]>();
  for (const { file, literals } of scanned) {
    for (const { line, value } of literals) {
      const locs = byValue.get(value) ?? [];
      locs.push({ file, line });
      byValue.set(value, locs);
    }
  }
  return byValue;
}

/** Formats one duplicated value and its locations into an error message. */
function formatDuplicate(value: string, locs: Location[]): string {
  const shown = locs.slice(0, MAX_LOCATIONS).map((l) => `${l.file}:${l.line}`);
  const more = locs.length - shown.length;
  const where = more > 0 ? `${shown.join(', ')}, +${more} more` : shown.join(', ');
  return `"${value}" appears ${locs.length} times (${where}) — extract a shared constant`;
}

/** One error message per non-allowed value that occurs >= MIN_OCCURRENCES times across all files. */
export function findDuplicates(
  scanned: { file: string; literals: Literal[] }[],
  allow: ReadonlySet<string> = new Set()
): string[] {
  const duplicated = [...groupByValue(scanned)].filter(
    ([value, locs]) => locs.length >= MIN_OCCURRENCES && !allow.has(value)
  );
  // Most-repeated first, then alphabetical by value for stable output.
  duplicated.sort((a, b) => b[1].length - a[1].length || (a[0] < b[0] ? -1 : 1));
  return duplicated.map(([value, locs]) => formatDuplicate(value, locs));
}

/** Fails when any string literal is repeated MIN_OCCURRENCES+ times across scanned source files. */
export const repeatedStringLiteralsCheck = {
  name: 'repeated-string-literals' as CheckName,
  async run(root = process.cwd()) {
    const files = await findSourceFiles(root);
    const scanned = files.map((file) => ({
      file,
      literals: scanStringLiterals(readFileSync(join(root, file), 'utf8')),
    }));
    const errors = findDuplicates(scanned, getAllowedValues(root));
    return checkResult(errors.length === 0, errors, files.length);
  },
  runInProcess: true,
} satisfies Check;

export default repeatedStringLiteralsCheck;

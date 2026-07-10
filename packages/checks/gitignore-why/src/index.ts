import { existsSync, readFileSync } from 'node:fs';
import { join } from 'node:path';
import { GITIGNORE, checkResult } from '@mayjournal/fitness-shared';
import type { Check, CheckName } from '@mayjournal/fitness';
import { enUS } from './enUS.js';

/** A classified .gitignore line: blank, a `#` comment, or an actual ignore pattern. */
export type LineKind = 'blank' | 'comment' | 'pattern';

/** Classify one raw .gitignore line: blank, comment (starts with `#`), or pattern (any other non-blank line). */
export function classifyLine(line: string): LineKind {
  const trimmed = line.trim();
  if (trimmed === '') return 'blank';
  if (trimmed.startsWith('#')) return 'comment';
  return 'pattern';
}

/** True when the line is a non-empty explanatory comment — a `#` followed by real text, not a bare `#`. */
export function isExplanatoryComment(line: string): boolean {
  const trimmed = line.trim();
  return trimmed.startsWith('#') && trimmed.replace(/^#+/, '').trim() !== '';
}

/** Format the violation message for a pattern line that lacks an explanatory comment directly above it. */
function formatViolation(lineNumber: number, text: string): string {
  return `.gitignore:${lineNumber}: pattern "${text}" ${enUS.NoComment}`;
}

/** Return a violation message for every pattern line not immediately preceded by an explanatory `#` comment. */
export function findViolations(content: string): string[] {
  const lines = content.split('\n');
  const errors: string[] = [];
  lines.forEach((line, index) => {
    if (classifyLine(line) !== 'pattern') return;
    const above = index > 0 ? lines[index - 1] : '';
    if (!isExplanatoryComment(above)) errors.push(formatViolation(index + 1, line.trim()));
  });
  return errors;
}

/** Requires every `.gitignore` pattern line to be immediately preceded by a `#` comment explaining why it exists. */
const gitignoreWhyCheck = {
  name: 'gitignore-why' as CheckName,
  async run(root = process.cwd()) {
    const path = join(root, GITIGNORE);
    if (!existsSync(path)) return checkResult(true, [], 0);
    const errors = findViolations(readFileSync(path, 'utf8'));
    return checkResult(errors.length === 0, errors, 1);
  },
  runInProcess: true,
} satisfies Check;

export default gitignoreWhyCheck;

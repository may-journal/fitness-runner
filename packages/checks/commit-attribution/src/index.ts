import { execSync } from 'node:child_process';
import { checkResult } from '@mayjournal/fitness-shared';
import type { Check, CheckName } from '@mayjournal/fitness';

const REQUIRED_TRAILERS = ['AI-Models', 'AI-Tools'] as const;

export const MSG_EMPTY =
  'No commit message to validate; add "AI-Tools:" and "AI-Models:" trailers to disclose AI usage';

/** Returns the first line (subject) of a commit message. */
function firstLine(s: string): string {
  return s.split('\n')[0] ?? '';
}

/** Returns true if the subject is an exempt commit (merge or revert). */
export function isExemptSubject(subject: string): boolean {
  return subject.startsWith('Merge ') || subject.startsWith('Revert ');
}

/** Returns true if the message has the trailer key with a non-empty value on its own line. */
export function hasTrailer(message: string, key: string): boolean {
  const re = new RegExp(`^${key}:[ \\t]*(\\S.*)$`, 'm');
  return re.test(message);
}

/** Returns an error message for each required AI-disclosure trailer that is missing. */
export function missingTrailers(message: string): string[] {
  const errors: string[] = [];
  for (const key of REQUIRED_TRAILERS) {
    if (!hasTrailer(message, key)) errors.push(`commit message missing "${key}:" trailer`);
  }
  return errors;
}

/** Resolves the full commit message from context or git log. */
function getCommitMessage(
  root: string,
  context: { proposedCommitMessage?: string } | undefined
): string {
  const proposed = context?.proposedCommitMessage;
  if (proposed !== undefined) return proposed;
  try {
    return execSync('git log -1 --pretty=%B', { cwd: root, encoding: 'utf8' });
  } catch {
    return '';
  }
}

/** Validates HEAD or proposed commit message discloses AI tools + models via git trailers. */
export const commitAttributionCheck: Check = {
  contextInline: { argName: '--message', contextKey: 'proposedCommitMessage' },
  name: 'commit-attribution' as CheckName,
  async run(root = process.cwd(), context) {
    const message = getCommitMessage(root, context);
    const subject = firstLine(message);
    if (!subject.trim()) return checkResult(false, [MSG_EMPTY], 1);
    if (isExemptSubject(subject)) return checkResult(true, [], 1);
    const errors = missingTrailers(message);
    return checkResult(errors.length === 0, errors, 1);
  },
};

export { REQUIRED_TRAILERS };

export default commitAttributionCheck;

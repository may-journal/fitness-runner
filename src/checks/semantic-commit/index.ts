import { execSync } from 'node:child_process';
import { createRequire } from 'node:module';
import { checkResult } from '../../utils/checkResult.js';

const require = createRequire(import.meta.url);
const conventionalCommitTypes = require('conventional-commit-types') as {
  types: Record<string, { description?: string }>;
};
import { CheckName } from '../../types/index.types.js';
import type { Check } from '../../types/index.types.js';

const SEMANTIC_TYPES = Object.keys(conventionalCommitTypes.types) as string[];
const SEMANTIC_RE = new RegExp(`^(${SEMANTIC_TYPES.join('|')})\\([^)]+\\): .+`);

/** Returns true if subject follows type(scope): description (or Merge commit). */
export function isSemanticSubject(subject: string): boolean {
  if (subject.startsWith('Merge ')) return true;
  return SEMANTIC_RE.test(subject);
}

export { SEMANTIC_TYPES };

export const MSG_EMPTY = 'No commit message to validate; use type(scope): description';

/** Resolves commit subject from context or git log. */
function getCommitSubject(
  root: string,
  context: { proposedCommitMessage?: string } | undefined
): string {
  if (context?.proposedCommitMessage !== undefined) return context.proposedCommitMessage;
  try {
    const msg = execSync('git log -1 --pretty=%B', { cwd: root, encoding: 'utf8' });
    return msg.split('\n')[0] ?? '';
  } catch {
    return '';
  }
}

/** Validates HEAD or proposed commit message follows Conventional Commits (type(scope): description). */
export const semanticCheck: Check = {
  contextInline: { argName: '--message', contextKey: 'proposedCommitMessage' },
  name: CheckName.SemanticCommit,
  async run(root = process.cwd(), context) {
    const subject = getCommitSubject(root, context);
    if (!subject.trim()) return checkResult(false, [MSG_EMPTY], 1);
    if (isSemanticSubject(subject)) return checkResult(true, [], 1);
    return checkResult(
      false,
      [
        `Commit message: "${subject}" — use type(scope): description (types: ${SEMANTIC_TYPES.join(', ')})`,
      ],
      1
    );
  },
};

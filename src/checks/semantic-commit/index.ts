import { execSync } from 'node:child_process';
import conventionalCommitTypes from 'conventional-commit-types' with { type: 'json' };
import type { Check } from '../../types/index.types.js';

const SEMANTIC_TYPES = Object.keys(conventionalCommitTypes.types) as string[];
const SEMANTIC_RE = new RegExp(`^(${SEMANTIC_TYPES.join('|')})\\([^)]+\\): .+`);

/** Returns true if subject follows type(scope): description (or Merge commit). */
export function isSemanticSubject(subject: string): boolean {
  if (subject.startsWith('Merge ')) return true;
  return SEMANTIC_RE.test(subject);
}

export { SEMANTIC_TYPES };

/** Resolves commit subject from context or git log. */
function getCommitSubject(
  root: string,
  context: { proposedCommitMessage?: string } | undefined
): string {
  if (context?.proposedCommitMessage) return context.proposedCommitMessage;
  try {
    const msg = execSync('git log -1 --pretty=%B', { cwd: root, encoding: 'utf8' });
    return msg.split('\n')[0] ?? '';
  } catch {
    return '';
  }
}

/** Validates HEAD or proposed commit message follows Conventional Commits (type(scope): description). */
export const semanticCheck: Check = {
  name: 'semantic-commit',
  async run(root = process.cwd(), context) {
    const subject = getCommitSubject(root, context);
    if (!subject) return { errors: [], meta: { filesChecked: 1 }, ok: true };
    if (isSemanticSubject(subject)) return { errors: [], meta: { filesChecked: 1 }, ok: true };
    return {
      errors: [
        `Commit message: "${subject}" — use type(scope): description (types: ${SEMANTIC_TYPES.join(', ')})`,
      ],
      meta: { filesChecked: 1 },
      ok: false,
    };
  },
};

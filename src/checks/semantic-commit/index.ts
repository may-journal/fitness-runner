import { execSync } from 'node:child_process';
import type { Check } from '../../types/index.js';

const SEMANTIC_TYPES = [
  'feat',
  'fix',
  'fixes',
  'docs',
  'style',
  'refactor',
  'test',
  'chore',
];
const SEMANTIC_RE = new RegExp(`^(${SEMANTIC_TYPES.join('|')})\\([^)]+\\): .+`);

/** Returns true if subject follows type(scope): description (or Merge commit). */
export function isSemanticSubject(subject: string): boolean {
  if (subject.startsWith('Merge ')) return true;
  return SEMANTIC_RE.test(subject);
}

export { SEMANTIC_TYPES };

/** Validates HEAD or proposed commit message follows Conventional Commits (type(scope): description). */
export const semanticCheck: Check = {
  name: 'semantic-commit',
  async run(root = process.cwd(), context) {
    const subject = context?.proposedCommitMessage ?? (() => {
      try {
        const msg = execSync('git log -1 --pretty=%B', { encoding: 'utf8', cwd: root });
        return msg.split('\n')[0];
      } catch {
        return '';
      }
    })();
    if (!subject) return { ok: true, errors: [], meta: { filesChecked: 1 } };
    if (isSemanticSubject(subject)) return { ok: true, errors: [], meta: { filesChecked: 1 } };
    return { ok: false, errors: [`Commit message: "${subject}" — use type(scope): description (types: ${SEMANTIC_TYPES.join(', ')})`], meta: { filesChecked: 1 } };
  },
};

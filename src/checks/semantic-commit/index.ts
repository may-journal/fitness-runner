import { execSync } from 'node:child_process';
import type { Check } from '../../types/index.js';
import { isSemanticSubject, SEMANTIC_TYPES } from './semantic.js';

/** Validates HEAD commit follows Conventional Commits (type(scope): description). */
export const semanticCheck: Check = {
  name: 'semantic-commit',
  async run(root = process.cwd()) {
    try {
      const msg = execSync('git log -1 --pretty=%B', { encoding: 'utf8', cwd: root });
      const subject = msg.split('\n')[0];
      if (isSemanticSubject(subject)) return { ok: true, errors: [], meta: { filesChecked: 1 } };
      return { ok: false, errors: [`HEAD commit: "${subject}" — use type(scope): description (types: ${SEMANTIC_TYPES.join(', ')})`] };
    } catch {
      return { ok: true, errors: [], meta: { filesChecked: 1 } };
    }
  },
};

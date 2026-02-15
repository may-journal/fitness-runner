const TYPES = [
  'feat',
  'fix',
  'docs',
  'style',
  'refactor',
  'test',
  'chore',
];
const SEMANTIC_RE = new RegExp(`^(${TYPES.join('|')})\\([^)]+\\): .+`);

/** Returns true if subject follows type(scope): description (or Merge commit). */
export function isSemanticSubject(subject: string): boolean {
  if (subject.startsWith('Merge ')) return true;
  return SEMANTIC_RE.test(subject);
}

export const SEMANTIC_TYPES = TYPES;

import { describe, it, expect } from 'vitest';
import { isSemanticSubject, SEMANTIC_TYPES } from './index.js';

describe('isSemanticSubject', () => {
  it('accepts type(scope): description', () => {
    expect(isSemanticSubject('feat(api): add endpoint')).toBe(true);
    expect(isSemanticSubject('fix(deps): bump cspell')).toBe(true);
    expect(isSemanticSubject('docs(readme): update links')).toBe(true);
  });

  it('accepts Merge commits', () => {
    expect(isSemanticSubject("Merge branch 'x' into main")).toBe(true);
  });

  it('rejects invalid format', () => {
    expect(isSemanticSubject('Fix something')).toBe(false);
    expect(isSemanticSubject('feat: no scope')).toBe(false);
    expect(isSemanticSubject('invalid(scope): missing type')).toBe(false);
  });

  it('exports SEMANTIC_TYPES', () => {
    expect(SEMANTIC_TYPES).toContain('feat');
    expect(SEMANTIC_TYPES).toContain('chore');
  });
});

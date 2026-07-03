/** Default check run order when no `.fitnessrc` `checks` list is set. */
export const defaultChecks = [
  'read-repo-first',
  'changelog',
  'changelog-updated',
  'cspell',
  'eslint',
  'markdown-no-bold-italic',
  'prettier',
  'node-version',
  'markdown-front-matter',
  'semantic-commit',
  'jscpd',
  'vitest-coverage-exclude',
  'vitest-coverage-full',
] as const;

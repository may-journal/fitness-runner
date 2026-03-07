/** Enum of all registered check names; use this instead of string literals. */
export enum CheckName {
  Changelog = 'changelog',
  ChangelogUpdated = 'changelog-updated',
  Cspell = 'cspell',
  Eslint = 'eslint',
  MarkdownFrontMatter = 'markdown-front-matter',
  MarkdownNoBoldItalic = 'markdown-no-bold-italic',
  NodeVersion = 'node-version',
  Prettier = 'prettier',
  ReadRepoFirst = 'read-repo-first',
  SemanticCommit = 'semantic-commit',
  VitestCoverageExclude = 'vitest-coverage-exclude',
  VitestCoverageFull = 'vitest-coverage-full',
}

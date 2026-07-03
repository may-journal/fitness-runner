/** Enum of defaultChecks-registered check names; use this instead of string literals. Opt-in checks that never join defaultChecks (e.g. swiftlint) intentionally aren't listed here — registry.test.ts enforces this enum stays exactly in sync with defaultChecks. */
export enum CheckName {
  Changelog = 'changelog',
  ChangelogUpdated = 'changelog-updated',
  Cspell = 'cspell',
  Eslint = 'eslint',
  Jscpd = 'jscpd',
  MarkdownFrontMatter = 'markdown-front-matter',
  MarkdownNoBoldItalic = 'markdown-no-bold-italic',
  NodeVersion = 'node-version',
  Prettier = 'prettier',
  ReadRepoFirst = 'read-repo-first',
  SemanticCommit = 'semantic-commit',
  VitestCoverageExclude = 'vitest-coverage-exclude',
  VitestCoverageFull = 'vitest-coverage-full',
}

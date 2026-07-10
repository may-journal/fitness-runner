/** @type {import('@mayjournal/fitness').FitnessConfig} */
const { defaultChecks } = require('@mayjournal/fitness-checks');

// Opt-in mermaid diagram + callout-table checks live in this repo, so we run them on our
// own architecture docs (dogfooding). They only activate on diagrams paired with a callout
// table, so unrelated diagrams are untouched.
const mermaidChecks = [
  'mermaid-callouts',
  'mermaid-callout-why',
  'mermaid-diagram-prose',
  'mermaid-legend',
  'mermaid-level-bleed',
];

module.exports = {
  // Every default check plus the opt-in mermaid + dependency-currency checks — the repo runs
  // the full suite on itself (dogfooding). Nothing is disabled.
  checks: [
    ...defaultChecks,
    ...mermaidChecks,
    'dependency-currency',
    'gitignore-why',
    // This repo's docs are kebab-case, so we enable the kebab-case flavor here (dogfooding);
    // the camelCase flavor stays opt-in (it would fail our hyphenated/numbered docs).
    'markdown-filename-kebab-case',
    'build-output-untracked',
    'repeated-string-literals',
  ],
  repeatedStringLiterals: {
    // Project baseline for repeated-string-literals (dogfooding). Everything here is a
    // structural repeat where a shared constant is impossible or worse than the literal;
    // genuine magic-string duplication stays flagged. Shrink this list, never grow it casually.
    allow: [
      // Check names: each necessarily appears in the bundler CHECK_SPECS, defaultChecks,
      // the CheckName enum, and/or this file. #41 (derive CHECK_SPECS from the filesystem)
      // would collapse most of these.
      'build-output-untracked',
      'changelog',
      'changelog-updated',
      'cspell',
      'dependency-currency',
      'eslint',
      'gitignore-why',
      'jscpd',
      'markdown-filename-kebab-case',
      'markdown-front-matter',
      'markdown-no-bold-italic',
      'mermaid-callout-why',
      'mermaid-callouts',
      'mermaid-diagram-prose',
      'mermaid-legend',
      'mermaid-level-bleed',
      'node-version',
      'prettier',
      'read-repo-first',
      'repeated-string-literals',
      'semantic-commit',
      'vitest-coverage-exclude',
      'vitest-coverage-full',
      // TS discriminated-union members: the union type declaration IS the closed set
      // (mermaid block kinds, gitignore-why LineKind); a constant cannot replace the
      // type-position occurrence.
      'diagram',
      'pattern',
      'table',
      // Declarative config values repeated across eslint/vitest/tsconfig configs and the
      // check sources that validate them (severities, rule ids, targets, globs, extensions).
      '**/*.d.ts',
      '**/*.spec.ts',
      '**/*.test.ts',
      '**/*.types.ts',
      '.cjs',
      '.cts',
      '.js',
      '.mts',
      '.ts',
      'ES2022',
      'error',
      'sort-keys',
      // Repo automation (scripts/, shared/bin, shared/config): standalone .cjs/.mjs that
      // cannot import the fitness-shared constants module (it is TS, built after they run).
      '--json',
      '../..',
      '.npmrc',
      '@mayjournal/fitness',
      '@mayjournal/fitness-shared',
      'CHANGELOG.md',
      'all',
      'build',
      'dist',
      'exit',
      'node_modules',
      'npm',
      'package.json',
      'packages/runner',
      'packages/shared/config',
      'run',
      'trust',
    ],
  },
};

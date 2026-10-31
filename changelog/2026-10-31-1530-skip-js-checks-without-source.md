### 2026.10.31.1530

- Feat: eslint and vitest-coverage-full pass with 0 files when a repo tracks no JavaScript or TypeScript source.
- Feat: a package.json used only for formatting tools no longer makes a repo a JavaScript project for those two checks.
- Feat: tool configs such as eslint.config.mjs and vitest.config.ts do not count as source.

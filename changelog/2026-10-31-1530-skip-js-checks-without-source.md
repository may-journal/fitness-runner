### 2026.10.31.1530

- Feat: eslint and vitest-coverage-full pass with 0 files when a repo tracks no JavaScript or TypeScript source.
- Feat: a package.json used only for formatting tools no longer makes a repo a JavaScript project for those two checks.
- Feat: tool configs named eslint, vitest, vite, jest, prettier, babel, rollup or webpack do not count as source.

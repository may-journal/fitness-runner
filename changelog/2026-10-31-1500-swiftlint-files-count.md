### 2026.10.31.1500

- Fix: swiftlint reports the Swift files it is asked to lint, not 0, on a pass or a fail.
- Feat: a pull request run counts only its changed Swift files, and a repo with none still reports 0.
- Feat: eslint and vitest-coverage-full pass with 0 files when a repo tracks no JavaScript or TypeScript source.
- Feat: a package.json used only for formatting tools no longer makes a repo a JavaScript project for those two checks.
- Feat: tool configs named eslint, vitest, vite, jest, prettier, babel, rollup or webpack do not count as source.

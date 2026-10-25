// Command fitness-check-build-output-untracked keeps build output out of git
// and out of source imports. It is on by default.
//
// # Build output is not tracked
//
// Rule A covers the `dist` directory and the output each language module
// names:
//
//   - `dist` must be git-ignored, as `git check-ignore -q dist` verifies. Fail hint: add `dist/` to `.gitignore`.
//   - `dist` must have no tracked files: `git ls-files -- dist` and `git ls-files -- '**/dist/**'` are both empty.
//   - The fail hint for tracked files: remove the listed files with `git rm --cached`.
//   - The build output each [language module] finds must have no tracked files either. They join the same list, in one combined error.
//
// # No imports from build output
//
// Rule B scans every tracked `.ts`, `.mts`, and `.cts` file, test and spec
// files included. It flags any `from '…'`, `import('…')`, or `require('…')`
// whose specifier reaches into a `dist/` path, such as `../dist/x.js`, and
// reports it as `path:line`.
//
// # Notes
//
//   - Git commands run from the repo root; only their stdout is parsed, so a failing git command contributes no phantom tracked files.
//   - `filesChecked` counts the scanned source files plus one for the dist git checks.
//   - Each module's package docs name its output paths. A repo with no such output and no `dist` rules to judge passes clean.
//
// [language module]: https://github.com/may-journal/fitness-runner/blob/main/go/internal/lang/langs/langs.go
package check

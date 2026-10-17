---
relatedConfigurations: ['../../../.fitnessrc.json']
---

# build-output-untracked

Keeps compiled build output out of Git and out of source imports. On by default.

## Behavior

- Rule A — build output not tracked (the `dist` directory):
  - `dist` must be git-ignored (verified with `git check-ignore -q dist`). Fail hint: add `dist/` to `.gitignore`.
  - `dist` must have no tracked files (`git ls-files -- dist` and `git ls-files -- '**/dist/**'` are both empty). Fail hint: remove the listed files with `git rm --cached`.
  - The build output each [language module](../../internal/lang/langs/langs.go) finds must have no tracked files either. They join the same list.
- Rule B — no imports from build output:
  - Scans every tracked `.ts`, `.mts`, and `.cts` file, test and spec files included.
  - Flags any `from '…'`, `import('…')`, or `require('…')` whose specifier reaches into a `dist/` path (e.g. `../dist/x.js`). Reported as `path:line`.

## Notes

- Git commands run from the repo root; only their stdout is parsed, so a failing git command contributes no phantom tracked files.
- `filesChecked` counts the scanned source files plus one for the dist git checks.
- Each module's package docs name its output paths. A repo with no such output and no `dist` rules to judge passes clean.

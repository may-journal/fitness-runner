---
relatedConfigurations: ['../../../.fitnessrc.json']
---

# build-output-untracked

Keeps compiled build output out of Git and out of source imports. Opt-in — not in the runner's default list. Add `"build-output-untracked"` to the `checks` array in `.fitnessrc.json` to enable it.

## Behavior

- Rule A — build output not tracked (the `dist` directory):
  - `dist` must be git-ignored (verified with `git check-ignore -q dist`). Fail hint: add `dist/` to `.gitignore`.
  - `dist` must have no tracked files (`git ls-files -- dist` and `git ls-files -- '**/dist/**'` are both empty). Fail hint: remove the listed files with `git rm --cached`.
- Rule B — no imports from build output:
  - Scans `.ts`, `.mts`, and `.cts` source files, excluding test/spec files (fixtures legitimately contain dist specifiers).
  - Flags any `from '…'`, `import('…')`, or `require('…')` whose specifier reaches into a `dist/` path (for example `../dist/x.js` or `foo/dist/x.js`). Each violation is reported as `path:line`.

## Notes

- Git commands run from the repo root; only their stdout is parsed, so a failing git command contributes no phantom tracked files.
- `filesChecked` counts the scanned source files plus one for the dist git checks.

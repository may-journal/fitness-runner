---
name: build-output-untracked
description: Fix a failing build-output-untracked check on a repo. Use when dist or other build output is tracked, not ignored, or imported.
---

# Keep build output out of Git

Compiled output must be ignored, untracked and never imported. [The check README](../../go/cmd/fitness-check-build-output-untracked/README.md) lists both rules.

## Read the failure

- `Add dist/ to .gitignore (git check-ignore reports dist is not git-ignored)`: add `dist/` to `.gitignore`.
- `Remove tracked files: dist/a.js, dist/b.js (git rm --cached)`: run `git rm --cached` on each listed file and keep them ignored.
- `src/a.ts:3 imports build output: "../dist/x.js"`: import from the source module instead.

## Check it first

Run `fitness build-output-untracked` from the repo root. Exit 0 means nothing is tracked or imported from build output.

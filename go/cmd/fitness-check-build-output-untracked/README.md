---
relatedConfigurations: ['../../../.fitnessrc.json']
---

# build-output-untracked

Keeps compiled build output out of Git and out of source imports. Opt-in — not part of `defaultChecks`. Add `'build-output-untracked'` to `.fitnessrc` `checks` to enable it.

## Behavior

- Rule A — build output not tracked (the `dist` directory):
  - `dist` must be git-ignored (verified with `git check-ignore -q dist`). Fail hint: add `dist/` to `.gitignore`.
  - `dist` must have no tracked files (`git ls-files -- dist` and `git ls-files -- '**/dist/**'` are both empty). Fail hint: remove the listed files with `git rm --cached`.
- Rule B — no imports from build output:
  - Scans `.ts`, `.mts`, and `.cts` source files (test/spec files excluded — fixtures legitimately contain dist specifiers) and flags any `from '…'`, `import('…')`, or `require('…')` whose specifier reaches into a `dist/` path (for example `../dist/x.js` or `foo/dist/x.js`). Each violation is reported as `path:line`.

## Notes

- Git commands run from the repo root via the shared `execSyncResult` helper (respects an injected exec for tests).
- `filesChecked` counts the scanned source files plus one for the dist git checks.
- Opt-in and bundled in `@mayjournal/fitness-checks/checks/*`, but never runs by default.

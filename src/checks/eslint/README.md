---
fitnessFunctions: ['eslint']
relatedConfigurations: ['../../../eslint.config.cjs']
---

# eslint

Runs [ESLint](https://eslint.org) for linting. Uses the project’s ESLint config (e.g. `eslint.config.js` or `package.json`).

## Behavior

- Pass: No errors or warnings in checked files.
- When context has staged files: Only those paths are linted.
- Otherwise: Runs `eslint .` from repo root.
- Lints `.ts`, `.tsx`, `.cjs`, `.js`, `.mjs` (excluding tests).
- Enforces `sort-keys` (natural ascending) for object literals.

Errors are reported as `path:line:col - message (ruleId)`.

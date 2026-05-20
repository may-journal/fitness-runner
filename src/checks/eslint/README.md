---
fitnessFunctions: ['eslint']
relatedConfigurations: ['../../../../packages/shared/config/eslint.config.cjs']
---

# eslint

Runs [ESLint](https://eslint.org) via this package's Node API and config. The parent project does not need ESLint, a local `eslint.config`, or a `tsconfig.json` installed; linting runs in the parent's working directory using this package's `eslint.config.cjs`. Type-aware rules use a temp tsconfig built from this package's compiler defaults scoped to the parent’s sources.

## Behavior

- Pass: No errors in checked files.
- Uses this package's ESLint and config; `cwd` is the parent project root.
- When context has staged files: Only those paths are linted; otherwise lints `.`.
- Lints `.ts`, `.tsx`, `.cjs`, `.js`, `.mjs` (excluding test files).
- Enforces `sort-keys` (natural ascending) and other rules from `eslint.config.cjs`.

Errors are reported as `path:line:col - message (ruleId)`.

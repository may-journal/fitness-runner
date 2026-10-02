---
fitnessFunctions: ['main.go']
relatedConfigurations: ['../../internal/sharedconf/config/eslint.config.mjs']
---

# eslint

Runs [ESLint](https://eslint.org) by executing the real `eslint` binary, resolved from `node_modules/.bin` (walking up from the repo root) then PATH — never npx. When no eslint binary resolves, the check fails with a one-line install hint. The shared flat config `eslint.config.mjs` is forced via `--config`.

The config resolves in order: the repo's own file, an installed `@mayjournal/fitness-shared` package under `node_modules`, then the copy embedded in the check binary. That embedded copy is materialized on demand.

The shared config imports its plugins (typescript-eslint, jsdoc, perfectionist, prettier) as bare specifiers. Those packages must be installed alongside eslint in the consumer repo — the check's peer contract. Linting is a syntactic parse only, with no type-aware rules.

## Behavior

- Pass: No errors in checked files.
- Execs eslint with `cwd` set to the repo root, forcing the resolved shared config via `--config`.
- In a scoped run: Only changed paths are linted, passing with 0 files when none is lintable. Otherwise lints every tracked lintable file.
- Lints `.ts`, `.tsx`, `.cjs`, `.js`, `.mjs`, declarations and tests included.
- Fail: A tracked file ESLint reports as ignored fails; remove the ignore pattern so it is linted.
- Enforces `sort-keys` (natural ascending) and other rules from `eslint.config.mjs`.

Errors are reported as `path:line:col - message (ruleId)`.

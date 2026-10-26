// Command fitness-check-eslint runs ESLint over the repo with the shared
// flat config, by executing the real [ESLint] binary. It resolves the binary
// from `node_modules/.bin`, walking up from the repo root, then PATH — never
// npx. When no eslint binary resolves, the check fails with a one-line
// install hint. The shared flat config `eslint.config.mjs` is forced via
// `--config`.
//
// The config resolves in order: the repo's own file, an installed
// `@mayjournal/fitness-shared` package under `node_modules`, then the copy
// embedded in the check binary. That embedded copy is materialized on demand.
//
// The shared config imports its plugins (typescript-eslint, jsdoc,
// perfectionist, prettier) as bare specifiers. Those packages must be
// installed alongside eslint in the consumer repo — the check's peer
// contract. Linting is a syntactic parse only, with no type-aware rules, so
// the runner-default timeout stands.
//
// # Behavior
//
//   - Pass: no errors in checked files.
//   - Execs eslint with `cwd` set to the repo root, forcing the resolved shared config via `--config`.
//   - In a scoped run: only changed paths are linted, passing with 0 files when none is lintable. Otherwise lints every tracked lintable file.
//   - Lints `.ts`, `.tsx`, `.cjs`, `.js`, `.mjs`, declarations and tests included.
//   - Fail: a tracked file ESLint reports as ignored fails; remove the ignore pattern so it is linted.
//   - Enforces `sort-keys` (natural ascending) and other rules from `eslint.config.mjs`.
//
// Errors are reported as `path:line:col - message (ruleId)`.
//
// # Biome
//
// A repo that chose Biome lints with it instead, as [ADR 0006] records. Such
// a repo has a `biome.json` or `biome.jsonc` at its root and `@biomejs/biome`
// in its `package.json` dependencies. The check runs
// `biome lint --reporter=github` on the same paths, and needs no eslint.
//
// Biome resolves like eslint, and a missing binary fails with an install
// hint. Biome before 1.8 fails naming its version, since its CLI rejects the
// GitHub reporter. Each Biome error or warning fails as
// `path:line:col - message (rule)`.
//
// [ESLint]: https://eslint.org
// [ADR 0006]: https://github.com/may-journal/fitness-runner/blob/main/docs/architecture/adr/0006-biome-stands-in-for-eslint-and-prettier.md
package check

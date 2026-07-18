---
relatedConfigurations: ['../../.fitnessrc.json']
---

# Plan: Repeating patterns in checks and abstraction options

This doc reviews `packages/checks/*` for recurring patterns and suggests shared abstractions to reduce duplication and keep behavior consistent.

## Result shape

Every check returns `CheckResult`: `{ errors: string[]; meta?: { filesChecked?: number }; ok: boolean }`.

- Many checks build this inline: `return { errors: [], meta: { filesChecked: 1 }, ok: true }` or variants with `filesChecked: 0`.
- Single-file or “single resource” checks (node-version, semantic-commit, vitest-coverage-exclude) often use `meta: { filesChecked: 1 }` on both success and failure.

Abstraction: A small helper, e.g. `checkResult(ok, errors?, filesChecked?)`, could normalize construction and default `meta.filesChecked` (e.g. 0 when omitted), so every check doesn’t repeat the same object shape.

## Context resolution

Checks repeatedly read from `RunContext` with the same fallbacks:

- `context?.stagedFiles ?? []`
- `context?._execSync ?? execSync`
- Optional: `proposedCommitMessage`, `enabledCheckNames`, `registeredCheckNames`, `passthroughArgs`

Examples: cspell `getContextExecAndStaged`, eslint `resolveInputs`, prettier `resolveInputs` / `getStaged`, changelog-updated `ensureChangelogExists` and `runChangelogUpdated`.

Abstraction: Shared helpers or a typed “resolved context” (e.g. `getStagedFiles(context)`, `getExecSync(context)`) so each check doesn’t reimplement the same fallbacks. Keep test-only keys like `_execSync` and `_now` as optional overrides in that layer.

## CLI-based checks: exec, parse, buildResult

eslint, prettier, and cspell (when using CLI) follow the same flow:

1. Build command and paths (or passthrough args).
2. `execSync` with shared options (encoding, maxBuffer).
3. On throw: extract `exitCode` and `output` from `err.status`, `err.stdout`, `err.stderr`.
4. Parse output into `errors` and optionally `filesChecked`.
5. Build result: `ok = exitCode === 0 && errors.length === 0`, with an optional fallback message when `!ok` but `errors.length === 0`.

Shared pieces: exec options, error extraction, and a common `buildResult(exitCode, errors, filesChecked, fallbackMessage?)`.

Abstraction: A small `runCliCheck` (or split into `execWithOpts`, `parseExecError`, `buildCliResult`) so each CLI check only supplies command building and output parsing. This would live in a shared util used by eslint, prettier, and cspell.

## Path resolution: staged vs default

- cspell: staged paths (existing) or default glob `**/*.md`.
- eslint: staged paths filtered by lintable extension and not test/spec, or `['.']`.
- prettier: staged paths with a skip set (e.g. `.gitignore`, `.husky/commit-msg`) or `['.']`.

So “staged vs default paths” and “filter/skip” are repeated with different rules.

Abstraction: A generic `getPathsToCheck(root, staged, options)` (or similar) with options such as `defaultPaths`, `extension`, `filter`, `skipSet` could centralize the “if staged use filtered staged, else use default” logic. Each check would pass its own filter/skip/defaults.

## File-by-file validation over a set of files

markdown-no-bold-italic and rules-front-matter both:

1. Get file list via `findFilesByExtension(root, '.md')`.
2. For each file: `readFileSync(join(root, file), 'utf8')`, run a `validateFile(file, content, ...)`.
3. Aggregate `errors` and `filesChecked`.

Only the validation function and extra args (e.g. `registeredCheckNames`) differ.

Abstraction: A helper like `runFileByFileCheck(root, extension, validateFile, options?)` that does the find, read, loop, and aggregation. Options could include passing `context` or extra arguments into the validator. This would reduce duplication in markdown-no-bold-italic and rules-front-matter.

## Config presence and early exit

- cspell: if `cspell.json` is missing, return `{ errors: [], meta: { filesChecked: 0 }, ok: true }`.
- prettier: if no Prettier config exists, return same.
- node-version: if `.nvmrc` is missing, return failure.

So “config file (or resource) missing” is handled differently: some checks skip (pass), some fail.

Abstraction: No single abstraction is required; documenting the convention (skip vs fail when config missing) in a checks README or this plan helps. Optionally, a tiny helper like `whenConfigMissing(path, root, { skip: true | false })` could return the appropriate early `CheckResult` so the convention is in one place.

## Reading and parsing JSON

changelog, vitest-coverage-exclude, and prettier read `package.json` (and sometimes lockfile or other config). They each do `existsSync` + `readFileSync` + `JSON.parse` with try/catch and handle invalid JSON.

Abstraction: A shared `readJsonFile(root, relativePath)` (or `readJsonAt(path)`) that returns parsed data or null on missing/invalid would remove repeated try/catch and existence checks. Typed wrappers (e.g. for package.json version or vitest config) can stay check-specific.

## Single-resource checks

node-version, semantic-commit, and vitest-coverage-exclude each validate one logical “resource” (one config file, one message, one config tree) and often use `filesChecked: 1` in the result.

Abstraction: A convention or helper for “single resource” checks (e.g. `singleResourceResult(ok, errors)` that sets `filesChecked: 1`) would keep result shape and semantics consistent without forcing all such checks into one implementation.

## Testability and injected dependencies

Checks that shell out or depend on time inject `_execSync` and `_now` via context so tests can avoid real exec and control time (e.g. changelog-updated, cspell).

Abstraction: Keep the current convention (optional `context._execSync`, `context._now`) and document it in a shared place (e.g. RunContext or a “Testing checks” section in the main checks README). No new abstraction required beyond documentation.

## Check display name and folder

read-repo-first uses `CHECK_TO_FOLDER` to map check name to folder path for display (e.g. `markdown-front-matter` → `rules-front-matter`). The registry is just an array of checks; folder is not part of the Check type.

Abstraction: Add an optional `folder?: string` (or `displayFolder`) to the Check type so the display logic can derive the README path from the check itself instead of a separate map. Default to `folder ?? name` so existing checks need no change.

## Summary table

| Completed                                          | Pattern                     | Checks involved                                        | Suggested abstraction                                  |
| -------------------------------------------------- | --------------------------- | ------------------------------------------------------ | ------------------------------------------------------ |
| [2026.02.22.1620](../../CHANGELOG.md#202602221620) | Result shape                | All                                                    | `checkResult(ok, errors?, filesChecked?)`              |
| [2026.02.22.1620](../../CHANGELOG.md#202602221620) | Context resolution          | cspell, eslint, prettier, changelog-updated            | `getStagedFiles`, `getExecSync`, resolved context type |
|                                                    | CLI exec/parse/buildResult  | eslint, prettier, cspell                               | `runCliCheck` or `execWithOpts` + `buildCliResult`     |
|                                                    | Staged vs default paths     | cspell, eslint, prettier                               | `getPathsToCheck(root, staged, options)`               |
|                                                    | File-by-file .md validation | markdown-no-bold-italic, rules-front-matter            | `runFileByFileCheck(root, '.md', validateFile)`        |
|                                                    | Config missing              | cspell, prettier, node-version                         | Convention + optional `whenConfigMissing`              |
|                                                    | Read JSON                   | changelog, vitest-coverage-exclude, prettier           | `readJsonFile(root, path)`                             |
|                                                    | Single-resource result      | node-version, semantic-commit, vitest-coverage-exclude | Convention or `singleResourceResult`                   |
|                                                    | Test inject exec/time       | changelog-updated, cspell                              | Document only                                          |
|                                                    | Check → folder for display  | read-repo-first                                        | Optional `Check.folder`                                |

Implementing these in order of impact (result helper and context resolution first, then CLI/path helpers, then file-by-file and JSON) would reduce duplication while keeping each check’s behavior and testability intact.

## Example: next pattern (CLI exec/parse/buildResult)

Shared util (e.g. `src/utils/cliCheck.ts`) could look like:

- `execWithOpts(root, command, execSyncFn)` — runs command with fixed `{ cwd: root, encoding: 'utf8', maxBuffer }`, returns `{ exitCode, output }` (on throw: parse status/stdout/stderr into exitCode and output).
- `buildCliResult(exitCode, errors, filesChecked, fallbackMessage?)` — returns `checkResult(ok, errors.length ? errors : (fallbackMessage ? [fallbackMessage] : []), filesChecked)` so each CLI check doesn't repeat the same fallback logic.

Then the eslint check would: resolve paths and exec via existing context helpers, call `execWithOpts(root, \`npx eslint ...\`, execFn)`, parse output with its existing `parseJsonResults`, and return `buildCliResult(exitCode, errors, filesChecked, ESLINT_FALLBACK_MESSAGE)`. Prettier and cspell (CLI path) would do the same with their own command builder and output parser.

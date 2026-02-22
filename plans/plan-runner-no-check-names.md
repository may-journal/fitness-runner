---
relatedConfigurations: ['../package.json']
---

# Plan: Abstract All Check-Name Logic from Runner (Config-Driven)

## Goal

Remove every reference to a specific check name (e.g. `semantic-commit`) from the runner. All check-specific behavior (e.g. "treat first extra positional as commit message file") will be driven by the fitness rc file so the runner stays generic.

## Current State

### Where check names are used in the runner

1. `getPassthroughArgs` (lines 34–46)
   - If `checkName === 'semantic-commit'`, the runner treats one positional as the commit-msg file path and excludes it from passthrough args.
   - Otherwise, all args after the check spec are passed through.

2. `getCommitMsgContext` (lines 48–65)
   - If `checkName === 'semantic-commit'`, the runner resolves a commit-msg file path from positionals (second when spec is positional, first when using `--check=`), reads the file's first line, and sets `proposedCommitMessage` on `RunContext`.
   - Otherwise, returns `undefined`.

3. `getChecks` (lines 159–167)
   - Passes `checkName` (when running a single check) into the two helpers above.

No other check names appear in the runner; the rest is already generic (registry, `config.checks`, etc.).

### Config today

- `FitnessConfig` (`src/types/fitness-config.types.ts`): only `checks?: string[]`.
- `.fitnessrc.ts` / `.fitnessrc.js`: used only to list which checks to run and in what order.

## Proposed Design

### 1. Extend config (rc file)

Add an optional map that describes "context file" behavior per check name:

- Option A (minimal): one context key, first extra positional = file path, first line → context.

  ```ts
  // In FitnessConfig
  /** When running a single check, optional map: checkName -> context file behavior. */
  checkContextFile?: Record<string, { contextKey: 'proposedCommitMessage' }>;
  ```

- Option B (extensible): allow more keys and optional read mode later.

  ```ts
  checkContextFile?: Record<string, {
    contextKey: keyof RunContext;  // e.g. 'proposedCommitMessage'
    readAs?: 'firstLine';          // default: first line of file
  }>;
  ```

Recommendation: Option A for now (only `proposedCommitMessage` is used). Option B can be adopted later if another check needs a different context key or read behavior.

So in `.fitnessrc.ts`:

```ts
export default {
  checks: ['changelog', 'semantic-commit', ...],
  checkContextFile: {
    'semantic-commit': { contextKey: 'proposedCommitMessage' },
  },
};
```

Commit-msg hook usage stays: `fitness --check=semantic-commit "$1"`; the runner uses config to know that for this check the first extra positional is a file whose first line becomes `proposedCommitMessage`.

### 2. Runner behavior (no check names in code)

- Resolve "context file" path
  Keep the existing positional rules (second positional when spec was positional, first when using `--check=`), but do not branch on `checkName`.
  Use a single helper, e.g. `getContextFilePath(positionals, checkNameIsFirstArg)`, that returns the appropriate positional when running a single check. The decision to use it is driven by config, not by name.

- Building context
  - Load config (already done in `resolveChecks`; ensure config is available where `getChecks` builds context).
  - When running a single check, look up `config?.checkContextFile?.[check.name]`.
  - If present: resolve context file path from positionals; if path is given, read file (first line), set `context[contextKey]` (e.g. `proposedCommitMessage`).
  - No `if (checkName === 'semantic-commit')` anywhere.

- Passthrough args
  When running a single check, if `config?.checkContextFile?.[check.name]` is set, resolve the same context file path and exclude it from passthrough args. Otherwise, pass all args through (current non–semantic-commit behavior).

- Config availability
  `getChecks` (or a shared helper) needs the resolved config. Today `resolveChecks(root)` loads config internally. Options:
  - Have `getChecks` call `loadConfig(root)` once and pass config into helpers that need it, or
  - Have a small "runner config" object (checks list + checkContextFile) built once in `getChecks` and passed through.
    Prefer a single `loadConfig(root)` at the start of `getChecks` and pass the result into `getCommitMsgContext`-replacement and `getPassthroughArgs`.

### 3. Naming in runner (internal only)

- Replace `getCommitMsgPath` / `getCommitMsgContext` with a single config-driven helper, e.g. `getContextFileFromConfig(config, checkName, argv, checkNameIsFirstArg)` that returns a `RunContext` fragment (e.g. `{ proposedCommitMessage }` or `{}`) and does not mention "commit" or "semantic" in logic—only "context file" and config keys.
- `getPassthroughArgs` becomes `getPassthroughArgs(config, argv, checkName, checkNameIsFirstArg)` and excludes the context file path only when `config?.checkContextFile?.[checkName]` is set.

Result: zero hardcoded check names in the runner; "semantic-commit" and commit-msg behavior exist only in user/config and docs.

### 4. Backward compatibility

- No default in runner: Do not hardcode a default `checkContextFile` for `semantic-commit` in the runner or config loader. So:
  - Repos that do not set `checkContextFile` and run `fitness --check=semantic-commit "$1"` would no longer get `proposedCommitMessage` from the file (breaking commit-msg hook until they add config).
- Migration: Document that for commit-msg hook behavior with semantic-commit, `.fitnessrc` must include:
  `checkContextFile: { 'semantic-commit': { contextKey: 'proposedCommitMessage' } }`.
- README / example: Update main README and any example `.fitnessrc` to show this entry so new and existing users get the expected behavior once they copy or add it.

Alternative (if you want zero config change for current users): support a default config merge only in the config loader (e.g. merge in a default `checkContextFile` for `semantic-commit` when user config does not set `checkContextFile`). Then the runner still has no `if (checkName === 'semantic-commit')`; the name appears only in the default config object. Prefer the explicit-config approach unless you need to avoid any change for existing repos.

### 5. Types

- `FitnessConfig`: add `checkContextFile?: Record<string, { contextKey: 'proposedCommitMessage' }>` (or the Option B shape if you prefer).
- `RunContext`: no change; already has `proposedCommitMessage`.

### 6. Tests

- Runner tests that rely on commit-msg behavior (e.g. `--check=semantic-commit` with a message file):
  - Ensure the test project has a `.fitnessrc` that includes `checkContextFile: { 'semantic-commit': { contextKey: 'proposedCommitMessage' } }` (and `checks` as needed).
  - Assert same behavior (exit 0/1, context passed to check) so the runner stays correct when config is present.
- New test: Running a single check not in `checkContextFile` does not treat any positional as a context file (no proposedCommitMessage, all positionals in passthrough).
- New test (optional): A check whose name is in `checkContextFile` receives `proposedCommitMessage` from the file and the file path is not in passthrough args.
- Remove any test that asserts "semantic-commit works with message file and no config" if we adopt the no-default approach (or keep it and expect failure / no context when config is missing).

### 7. Docs

- README: Commit-msg hook section: state that `checkContextFile` must be set for semantic-commit (e.g. `checkContextFile: { 'semantic-commit': { contextKey: 'proposedCommitMessage' } }`) and show a full example.
- semantic-commit check README: Mention that the runner uses rc config to read the message file; point to main README or config docs for `checkContextFile`.
- Flow diagram (if any): Replace "getCommitMsgContext if --check=semantic-commit + positional path" with "context file from config + positionals" or similar.

## Implementation Order

1. Types: Extend `FitnessConfig` with `checkContextFile`.
2. Runner: Introduce `getContextFilePath(positionals, checkNameIsFirstArg)` (or equivalent) and config-driven helpers; replace `getCommitMsgPath` / `getCommitMsgContext` and the `semantic-commit` branches in `getPassthroughArgs` with config lookups.
3. Tests: Add/update runner tests to use `.fitnessrc` with `checkContextFile` for semantic-commit; add tests for "no context file when not in config" and "context file applied when in config."
4. Docs: Update README, semantic-commit README, and any Mermaid diagram.
5. Changelog: Entry describing the config-driven context file and the new `checkContextFile` option; note migration for commit-msg hook users if we do not ship a default.

## Summary

| Item                         | Action                                                                    |
| ---------------------------- | ------------------------------------------------------------------------- |
| Runner check-name branches   | Remove all; use only `config.checkContextFile?.[check.name]`.             |
| Config                       | Add `checkContextFile` in `.fitnessrc` (per-check context file behavior). |
| Commit-msg / semantic-commit | Document and rely on rc; no "semantic-commit" in runner code.             |
| Tests                        | Use rc in tests; add coverage for config-driven context file.             |
| Docs                         | README + check README + diagram updated for config-driven behavior.       |

---
# Changed package version should correlate with this file
relatedConfigurations: ['package.json']
---

# Changelog

## Changes

### 2026.03.07.1007

- Cspell: add runCspell (CLI runner), enUS dict, runCspell.test; refactor check to runViaExec/runViaLib; add JSDoc and reduce complexity. Utils: add isMainModule(import.meta.url) and tests. Plans: update plan-checks-abstractions.md.

### 2026.02.22.1620

- Checks: add checkResult(ok, errors?, filesChecked?) and runContext (getStagedFiles, getExecSync); migrate all checks to use them. RunContext gains `_now` for tests.
- Plans: add plan-checks-abstractions.md (repeating patterns in checks/\*, abstraction options). README: flowchart node renamed to PassthroughArgs. cspell.json: trim words list.
- Dependencies: flatten into dependencies only (no dev/optional). Prettier: package.json override to use json parser so sort-json runs recursively (exports paths and condition keys); remove prettier-plugin-packagejson; add comments in prettier.config.cjs.
- Runner: enUS enum for all user-facing copy (runner/enUS.ts); run logic in run.ts, index.ts barrel only. Add interpolate() util for {{key}} templates; total line uses enUS.TotalLine. Export enUS from runner and package. ESLint: eslint-plugin-typescript-sort-keys (string-enum + interface), @typescript-eslint aligned to ^8.55; overrides for plugin eslint peer.

### 2026.02.22.1511

- Prettier: single config (prettier.config.cjs), remove .prettierrc.json; add .prettierignore and prettier.config.d.ts with package.json types export; Prettier check recognizes .ts/.mts/.cts config names; .gitignore generated Prettier files; drop unsupported ignore option from config.

### 2026.02.22.1454

- semantic-commit: fail when no message to validate (empty or git unavailable). commit-msg hook: pass message content via --message="$(cat "$1")". Export `MSG_EMPTY`.

### 2026.02.22.1431

- Utils: replace findMd with findFilesByExtension(root, extension). getSkipDirs uses skipTheseDirectories from .fitnessrc if present, else cspell.json ignorePaths (dir names only); no default list. Add FitnessConfig.skipTheseDirectories; add .git, .husky to cspell.json.

### 2026.02.22.1408

- Checks: add CheckName enum; all checks use enum for name (no static strings). Export CheckName from types; runner casts config check names to CheckName for registry lookup.

### 2026.02.22.1359

- Docs: make Mermaid flowchart edge labels readable (linkStyle color for yes/no arrows).

### 2026.02.22.1332

- Runner: check-registered contextInline; no check-name logic. Check type gains optional contextInline (argName, contextKey); semantic-commit registers --message → proposedCommitMessage; commit-msg hook uses --message="$(cat \"$1\")". Export ContextInline; update README and flow diagram.

### 2026.02.22.1322

- Plan: abstract runner check-name logic via check-registered context (contextInline/contextPath on Check type; no .fitnessrc change).

### 2026.02.22.1259

- Runner: rename specFromPositional to checkNameIsFirstArg and getCommitMsgPath to getContextFilePath for clarity; add plans/plan-runner-no-check-names.md with front matter and no bold/italic for checks.

### 2026.02.22.1247

- Runner: abstract check deps; move getColumns to src/utils/terminal so runner has no check-specific imports; read-repo-first no longer exports getColumns; getColumns tests moved to utils/terminal.test.ts.

### 2026.02.22.1241

- README: align Mermaid code-flow diagram with runner (resolveCheckSpec, spec-defined branch, resolveChecksBySpec, buildContext).

### 2026.02.16.1646

- Changelog check: require package.json version suffix and package-lock.json version to match CHANGELOG first ### heading (yyyy.mm.dd.HHMM).

### 2026.02.16.1634

- Changelog-updated check: require new section heading to use current date and time (yyyy.mm.dd.HHMM) so GenAI cannot guess the time.

### 2026.02.16.1900

- Changelog-updated check: export human-facing message consts `(MSG_*)`; reuse in implementation and tests.

### 2026.02.16.1800

- Runner: add passthroughArgs to RunContext when running a single check; args after check name forwarded to checks (e.g. `npx fitness prettier --write`).
- Prettier check: use passthroughArgs; run Prettier with forwarded args instead of --check when present.

### 2026.02.16.1700

- Package: move eslint, prettier, vitest, and related config plugins from devDependencies to dependencies so consumers can use exported configs; keep @types/node, tsconfig.js, tsx, typescript as devDependencies.
- Prettier check: detect config via package.json "prettier" field so consumers using `"prettier": "@fitness/runner/prettier.config"` are checked.

### 2026.02.16.1600

- ESLint: add sort-keys rule (natural ascending) for all object keys in .ts, .cjs, .js, .mjs; extend ESLint check to .cjs/.js/.mjs; reorder object literals across codebase.
- Prettier: add prettier-plugin-sort-json with jsonRecursiveSort for alphabetical ordering of object keys and nested objects in arrays (JSON files).
- Prettier check: runs prettier --check; skips when no config; staged files or full repo; exports prettierCheck.
- Prettier: add Prettier with eslint-config-prettier; .prettierrc.json and .prettierignore; format/format:check scripts; CI format job; export prettier.config for consumers.
- README: sync main and checks README with registry; add rules-front-matter README.
- ESLint check README: add fitnessFunctions; relate to eslint.config.cjs.
- Runner: table feedback with colspan row per failed check; errors contextual to row; dynamic width via getColumns from read-repo-first; README flow diagram update.
- Rules-front-matter: reject empty fitnessFunctions and relatedConfigurations arrays; require at least one entry per array.

### 2026.02.16.1500

- Runner: format check results as table (cli-table3) with Check, Status, Files, Time columns; bold white headers.
- Read-repo-first: table of enabled checks with Src column (plain paths for IDE link detection); add cli-table3.
- Read-repo-first: remove TTY requirement; display feedback to CLI for Agent/User context; always pass. Remove `FITNESS_READ_REPO_CI_ONLY_DO_NOT_USE_OTHERWISE` from CI.
- Read-repo-first check: prompts Y/N to confirm familiarity with Fitness Checks; lists enabled checks; chalk/boxen/wrap-ansi formatting; runs first in registry.
- Runner: chalk formatting for check results (green/red), errors, total line; `FITNESS_READ_REPO_CI_ONLY_DO_NOT_USE_OTHERWISE` bypass for CI.
- Cursor rules: consolidate into fitness-checks.mdc; remove changelog, changelog-updated, node-version, semantic-commit, vitest-coverage-exclude rules.
- ESLint: add eslint.config.d.ts for ESM package compatibility.
- Package and gitignore: updates for consolidated rules.

### 2026.02.16.1400

- Package: add exports for eslint.config, vitest.config, tsconfig, tsconfig.cjs, and cspell for reuse by downstream projects.

### 2026.02.16.1300

- Vitest-coverage-exclude: update README to use relative paths; properly associated with the name of the check.
- Tsconfig: single root tsconfig.cjs for build and lint; remove build/; use tsconfig.js to convert .cjs to JSON.
- Changelog-updated: ExecSyncFn maxBuffer type; add execSync fallback coverage test.
- ESLint: replace eslint.config.ts with eslint.config.cjs for ESM package compatibility.
- Tsconfig: remove shared base config; inline compiler options in tsconfig.cjs.
- ESLint: remove stylistic plugin and rules; keep jsdoc/require-jsdoc and complexity max 5. Single block, project tsconfig.json.
- Gitignore: ignore compiled root config outputs (eslint and vitest .js, .map, .d.ts).

### 2026.02.16.1025

- ESLint check: run eslint (staged or .), parse JSON, report errors; hoist feedback and CLI consts; 100% coverage.
- ESLint check: only pass .ts/.tsx staged paths (avoid no-config for .md); tests use .ts (bar, pathWithQuote, quoted).

### 2026.02.16.1005

- Runner: add feedback dressing (Please fix these items), hoist messages to shared consts; static import in runner tests.

### 2026.02.16.0958

- Plans: add read-repo-first check design doc (plans/read-repo-first-check.md).

### 2026.02.15.1700

- Docs: clarify README tagline (fitness runner, checks, workflows).
- Cspell: run in-process via cspell-lib (readConfigFile, spellCheckFile) for speed; keep CLI path when tests mock exec.
- Runner: add total files scanned count to summary (sum of filesChecked from checks).
- Runner: print total success and failure count and round time (performance) after runs.
- Markdown-no-bold-italic: do not flag unordered list asterisk markers as italic.

### 2026.02.15.1600

- Docs: remove bold/italic from check READMEs to satisfy markdown-no-bold-italic.
- Remove .fitnessrc.ts.
- Runner test: cover resolveChecks when .fitnessrc.ts provides custom checks (restore 100% coverage).
- Rules-front-matter: allow fitnessFunctions and relatedConfigurations to reference any registered check name (not just paths).
- Vitest-coverage-exclude check: only allow `**/*.d.ts` and `**/*.types.ts` in coverage exclude; Vitest excludes tests by default. Type-only files use `*.types.ts` naming.
- Rename type files to `*.types.ts`; fix load.ts and coverage-exclude branches for 100% coverage.
- CI: single fitness job runs npm run fitness; remove discover job and matrix.
- Pre-commit: source nvm (`NVM_DIR`, nvm.sh) in husky hook so nvm use runs when PATH has no nvm.
- CI: list checks as single-line `GITHUB_OUTPUT` (printf, tr -d newline) to avoid EOF delimiter; valid JSON for fitness job matrix.
- Markdown-front-matter: require fitnessFunctions or relatedConfigurations in every .md; paths resolved relative to md file; findMd skips node_modules, dist, coverage, .git, .husky; export getFrontMatterPaths for tests (100% coverage). README/CHANGELOG front matter fixes (---, flow-style).

### 2026.02.15.1500

- Node-version check: compare .nvmrc to current Node only; remove nvm subshell logic. CI script runs nvm use when available.
- Node-version check: run nvm use in a subshell when available and use that version for validation; fallback to process.version.
- CI: default strategy matrix fromJson(needs.discover.outputs.checks) to '[]' when checks output is empty.

### 2026.02.15.1400

- Runner: 100% coverage; path-load tests (named export, no Check, import throws); two positionals (check then msg path) for semantic-commit; getPositionalSpec and getCommitMsgContext fix for single vs two positionals.

### 2026.02.15.1300

- Rules front-matter check: validate fitnessFunctions and relatedConfigurations paths in all markdown; shared findMd helper.
- Runner: accept check by name or path (--check=./path/to/check.js or positional); load Check from module default or named export.

### 2026.02.15.0100

- Runner: single CLI flag --check= only; commit-msg path positional; staged context always; full runner test coverage.

### 2026.02.15.1200

- GitHub Actions CI: dynamic fitness jobs from registry, composite setup action.
- README: add Mermaid code-flow diagram (modern colors), move to bottom.
- cspell check (optional: only runs when cspell.json present); cspell in runner dependencies.
- cspell in deps only; remove spell from ci and package.json script; README and cspell README updates.

### 2026.02.15.1100

- Commit-msg hook for semantic-commit; merge semantic-commit into single file; cursor rules point to check READMEs.
- Changelog check: require ### yyyy.mm.dd.HHMM; every ### heading must match.
- Remove duplicate check-node-version script and check-node; ci runs npm run fitness only.
- Changelog-updated check: suggest up to 10 random words from staged diff when overlap is too low.
- Changelog section format: yyyy.mm.dd.HHMM to match package version.

### 2026.02.15.1000

- Add fitness config (.fitnessrc.ts) and config loader; turn on all checks (changelog, semantic-commit).
- Changelog-updated check (fuzzy match staged diff to changelog); colocate tests with source; merge runner tests, only checking changed lines, not whole file.

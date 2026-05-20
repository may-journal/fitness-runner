---
# Changed package version should correlate with this file
relatedConfigurations: ['package.json']
---

# Changelog

## Changes

### 2026.05.20.1509

- ESLint: centralize rules in `eslint.base.cjs`; CLI and fitness eslint check import `createEslintConfig` with their own TypeScript parser options so `projectService` and `project` no longer conflict.
- Shared bin: add `fitness-shared lint` for monorepo ESLint; wire workspace lint scripts through it; run lint in pre-commit alongside fitness.
- Check packages: add minimal lint-only `tsconfig.json` (extends shared check config) for ESLint `projectService` discovery; remove monorepo check enumeration from shared config.

### 2026.05.20.1453

- Shared config: centralize TypeScript `compilerOptions` in `tsconfig.compiler.cjs`; generate `tsconfig.check.json` and `tsconfig.checks.json` from CJS sources so ESLint and build share one source of truth.
- CI: commit hand-authored check and bundle `tsconfig.json` files (un-ignore in `.gitignore`) and build `@mayjournal/fitness-shared` before other workspaces so runner `tsc` resolves shared types.
- ESLint: include check package and bundle `tsconfig.json` paths in `parserOptions.project` so type-aware lint finds monorepo check sources.
- Cspell: skip staged `.gitignore`, `package-lock.json`, and `tsconfig.json` so explicit staged paths honor `ignorePaths`.
- Restore `disabledChecks` on `.fitnessrc`: optional list removes names from an explicit `checks` list or from bundle `defaultChecks` after `resolveCheckNames`; types on `@mayjournal/fitness-shared`, filter in runner `load-check.ts`; tests in `load-check.test.ts` and `run.test.ts`.
- Add plans/plan-split-runner-check-packages.md for splitting @mayjournal/fitness runner from per-check npm packages.
- Mark plan step 7d release with PR #12 in plans/plan-split-runner-check-packages.md.
- Move runner and checks from root src/ into packages/runner and twelve packages/checks/\* workspaces with dynamic check loading and @mayjournal/fitness-checks-bundle defaults.
- Add Architecture.md for monorepo layout; update README and complete plan steps 1–6 (shared configs, per-check vitest, remove legacy src/).
- Scaffold npm workspaces: private root package.json, packages/runner, packages/shared, packages/checks-bundle, and packages/checks/\* stubs; bump versions under packages/ in ensure-changelog-timestamp.cjs; ignore **/coverage/** in cspell.
- Move check tool dependencies from packages/runner into each packages/checks/\* package so published check packages declare only what they need; trim runner deps and refresh package-lock.json.
- CI: run `npm run test -ws --if-present` so every workspace with a test script runs in GitHub Actions.
- Check packages: scope Vitest to `src/**/*.test.ts`, point `@mayjournal/fitness` aliases at runner `dist/types`, and give vitest-coverage-full an explicit coverage config.
- Changelog-updated: expect new section heading to match root package.json version suffix so pre-commit timestamp bumps pass after long CI runs.
- Checks fall back to @mayjournal/fitness configs when consumers lack local cspell, prettier, vitest, or tsconfig; add resolveFitnessConfigPath, resolveLintTsconfig (temp tsconfig for ESLint in parent cwd), and tsconfig.lint.cjs export; document consumer setup in README.
- Switch publish workflow to npm trusted publishing (OIDC); use NODE_AUTH_TOKEN in .npmrc instead of NPM_TOKEN secret.

### 2026.04.04.1748

- Export `./vitest.config` as `vitest.config.mjs` and add `vitest.config.d.ts` for TypeScript consumers.
- Resolve Prettier plugin paths with `createRequire` so consumers loading `@mayjournal/fitness/prettier.config` resolve plugins from this package.

### 2026.04.04.1712

- ESLint check: run via Node API with this package's eslint.config.cjs and parent project cwd so consumers need not install ESLint; add getFitnessRunnerRoot util; eslint runInProcess; RunContext \_eslintRunForTesting for tests; runner and vitest-coverage-full use shared root helper.
- ESLint check: show file:line:col errors by extracting JSON array when stderr is mixed in; on parse failure append truncated ESLint output to fallback; add tryParseJsonArray to keep complexity under limit.
- Scope package to @mayjournal/fitness; add LICENSE (MIT), GitHub Actions publish workflow (on CI success), publish:ci script, .npmrc for NPM_TOKEN; add plan-deps-vs-devdeps-check.md.

### 2026.03.07.1922

- Prettier: use prettier-plugin-packagejson for conventional package.json field order; add package-lock.json to .prettierignore so Prettier does not touch it.
- Refactor: single source of truth for check registration. Add optional `Check.folder` and `RunContext.checkFolderByName` (serializable for worker); runner builds map from registry. Remove `CHECK_TO_FOLDER` from read-repo-first; rules-front-matter sets `folder: 'rules-front-matter'`. Add registry.test.ts to assert `CheckName` enum and registry stay in sync; document add-a-check steps in src/checks/README.md.
- Refactor: shared quoteForShell in src/utils/shellQuote.ts; use in eslint, prettier, cspell. Shared Vitest config loader in src/checks/vitest-config (`VITEST_CONFIG_NAMES`, loadVitestConfig, getCoverageExcludeFromConfig, getThresholdsFromConfig); vitest-coverage-exclude and vitest-coverage-full use it.
- Refactor: shared exec helper and buildExecCheckResult for CLI checks. Add execSyncResult() and EXEC_OPTS in src/utils/execSyncResult.ts; add buildExecCheckResult() in checkResult.ts. Use in eslint, prettier, cspell, vitest-coverage-full, changelog-updated.
- Markdown-no-bold-italic: ignore emphasis inside link blocks [text](url) so underscores in URLs or link text are not falsely flagged.

### 2026.03.07.1431

- Runner: dedupe config.checks by name so each check runs once when .fitnessrc lists the same check multiple times.

### 2026.03.07.1406

- ESLint: add max-lines rule (200, skipBlankLines/skipComments). Runner: split run.ts into run-resolve.ts (getChecks, config/spec resolution), run-execute.ts (runOneCheck, worker/in-process), run-output.ts (buildTable, buildTotalLine); run.ts keeps orchestration only.

### 2026.03.07.1007

- Runner: always ignore `node_modules`—getSkipDirs returns runner skip dirs (`node_modules`, dist, coverage, .git, .husky) merged with config; staged files from git are filtered to exclude paths under `node_modules`.
- Runner: run registry checks in a worker thread so 5s timeout is enforced via worker.terminate() when checks block (e.g. execSync); read-repo-first, vitest-coverage-full, and path-based checks stay in-process. Add run-one-check-worker.ts; tests use in-process (VITEST).
- Vitest-coverage-exclude: add vitest.config.cjs to `VITEST_CONFIG_NAMES`.
- Runner: add 5s per-check timeout; timed-out checks fail with "Check timed out after 5s" and runner continues.
- Runner: add progress messages to stderr (Resolving checks…, Running checks:, and → name before each check) so users can see where the run is or where it hangs.
- Utils: fix isMainModule when run via npx (resolve argv[1] and import.meta.url to real paths so symlinked .bin/fitness is detected as main). Add symlink test; add JSDoc and reduce complexity for lint.
- Checks: add vitest-coverage-full (runs vitest run --coverage; requires 100% thresholds in consumer and @mayjournal/fitness package). Vitest-coverage-exclude: allow barrel index.ts exclude pattern. Add vitest.config.mjs (ESM). Cspell: add runCspell (CLI runner), enUS dict, runCspell.test, word unstub; refactor check to runViaExec/runViaLib; add JSDoc and reduce complexity. Utils: add isMainModule(import.meta.url) and tests. Plans: update plan-checks-abstractions.md.

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
- Prettier check: detect config via package.json "prettier" field so consumers using `"prettier": "@mayjournal/fitness/prettier.config"` are checked.

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
- Markdown-front-matter: require fitnessFunctions or relatedConfigurations in every .md; paths resolved relative to md file; findMd skips `node_modules`, dist, coverage, .git, .husky; export getFrontMatterPaths for tests (100% coverage). README/CHANGELOG front matter fixes (---, flow-style).

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

---
# Changed package version should correlate with this file
relatedConfigurations: ["package.json"]
---

# Changelog

## Changes

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
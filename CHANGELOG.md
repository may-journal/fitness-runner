# Changelog

## Changes

### 2026.02.15.1300
- Rules front-matter check (30.03): validate fitnessFunctions and relatedConfigurations paths in all markdown; shared findMd helper.
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
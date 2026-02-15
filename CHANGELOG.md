# Changelog

## Changes

### 2026-02-15

- Add fitness config (.fitnessrc.ts) and config loader; turn on all checks (changelog, semantic-commit).
- Changelog-updated check (fuzzy match staged diff to changelog); colocate tests with source; merge runner tests, only checking changed lines, not whole file.
- Commit-msg hook for semantic-commit; merge semantic-commit into single file; cursor rules point to check READMEs.

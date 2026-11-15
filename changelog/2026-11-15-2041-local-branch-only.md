### 2026.11.15.2041

> The engine names `local-branch` directly, and this repo keeps no check that guards only code that moved to fitness-checks.

- Refactor: `fitness-release checks` reads only `local-branch` and writes one entry point, summary and doc in place of three maps; unused release `Config.asset` goes.
- Docs: `docs/checks.md` and the architecture views model only the engine, its hooks, `local-branch` and the checks it installs.
- Chore: the local checks `adr-0002`, `adr-0006` and `adr-0010` are retired, and `adr-0009` keeps only its `.fitnessrc.json` rule.
- Test: requirement 0114 covers the README command list alone, and 0161 names `checks.go` as the one file that links a check.
- Test: 26 unreachable requirement test helpers and nine constants nothing names are deleted; the `edited` helper and `prEnv` move to `repos_test.go`.

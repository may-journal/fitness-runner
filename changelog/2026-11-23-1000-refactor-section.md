### 2026.11.23.1000

> Refactor commits now count as user-facing changes, so Release Please opens a release PR for them.

- Build: `release-please-config.json` sets `changelog-sections` for the root package.
- Build: `refactor` is a visible "Refactors" section beside feat, fix, perf and revert.
- Build: chore, docs, test, build and ci stay hidden, as in the defaults.

---
relatedConfigurations: ['../../internal/sharedconf/config/vitest.config.mjs']
---

# vitest-coverage-exclude

## Behavior

- Pass: No Vitest config, or `coverage.exclude` is empty or missing.
- Fail: Any exclude entry, tests and type files included → one error per entry, naming it to remove.
- Vitest's own default exclude skips tests and declarations, so set `coverage.exclude: []` to measure them; the shared config does.

Reads `vitest.config.*` or `package.json` `vitest` in the project root. If none exist, it uses the shared vitest config. That is an installed `@mayjournal/fitness-shared` package when present, else the copy embedded in the check binary, materialized on demand.

## Changelog for this file

- 2026-02-15: Initial rule

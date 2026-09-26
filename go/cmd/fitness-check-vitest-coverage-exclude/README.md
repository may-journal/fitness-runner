---
relatedConfigurations: ['../../internal/sharedconf/config/vitest.config.mjs']
---

# vitest-coverage-exclude

## Behavior

- Pass: No Vitest config, or `coverage.exclude` is empty or missing, or every entry ends with `.d.ts`, `.types.ts`, `.test.ts`, or `.spec.ts`.
- Fail: Any exclude entry is a non-conventional .ts pattern (e.g. `src/foo.ts`, `src/types/**`) → error listing each disallowed pattern.

Reads `vitest.config.*` or `package.json` `vitest` in the project root. If none exist, it uses the shared vitest config. That is an installed `@mayjournal/fitness-shared` package when present, else the copy embedded in the check binary, materialized on demand.

## Changelog for this file

- 2026-02-15: Initial rule

---
relatedConfigurations: ['../../internal/sharedconf/config/vitest.config.mjs']
---

# vitest-coverage-exclude

## Behavior

- Pass: No Vitest config (falls back to the shared vitest config), or `coverage.exclude` is empty/missing, or every exclude entry ends with `.d.ts`, `.types.ts`, `.test.ts`, or `.spec.ts` (e.g. `src/**/*.types.ts`).
- Fail: Any exclude entry is a non-conventional .ts pattern (e.g. `src/foo.ts`, `src/types/**`) → error listing each disallowed pattern.

Reads `vitest.config.*` or `package.json` `vitest` in the project root. If none exist, it uses the shared vitest config — an installed `@mayjournal/fitness-shared` package when present, else the copy embedded in the check binary, materialized on demand.

## Changelog for this file

- 2026-02-15: Initial rule

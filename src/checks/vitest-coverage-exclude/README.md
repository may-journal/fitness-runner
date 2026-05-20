---
relatedConfigurations: ['vitest-coverage-exclude']
---

# vitest-coverage-exclude

## Behavior

- Pass: No Vitest config (falls back to `@mayjournal/fitness` vitest config), or `coverage.exclude` is empty/missing, or every exclude entry ends with `.d.ts`, `.types.ts`, `.test.ts`, or `.spec.ts` (e.g. `src/**/*.types.ts`).
- Fail: Any exclude entry is a non-conventional .ts pattern (e.g. `src/foo.ts`, `src/types/**`) → error listing each disallowed pattern.

Reads `vitest.config.*` or `package.json` `vitest` in the project root; if none exist, uses this package's vitest config.

## Changelog for this file

- 2026-02-15: Initial rule

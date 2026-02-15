---
relatedConfigurations: ["../../../vitest.config.ts", "../../../package.json"]
---

# vitest-coverage-exclude

## Behavior

- Pass: No Vitest config, or `coverage.exclude` is empty/missing, or every exclude entry ends with `.d.ts`, `.types.ts`, `.test.ts`, or `.spec.ts` (e.g. `src/**/*.types.ts`).
- Fail: Any exclude entry is a non-conventional .ts pattern (e.g. `src/foo.ts`, `src/types/**`) → error listing each disallowed pattern.

Reads `vitest.config.ts`, `vitest.config.js`, or `package.json` `vitest` key.

## Changelog for this file

* 2026-02-15: Initial rule

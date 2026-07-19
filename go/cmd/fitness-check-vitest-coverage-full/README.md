---
relatedConfigurations: ['../../internal/sharedconf/config/vitest.config.mjs']
---

# vitest-coverage-full

## Behavior

- Pass requires all three of:
  - The project sets Vitest coverage thresholds to 100 for branches, functions, lines, and statements.
  - The fitness-runner shared config carries the same 100% thresholds.
  - `vitest run --coverage` exits 0 in the project root.
- Both configs are checked statically. Neither side may lower thresholds to pass.
- Threshold and exclude settings come from the project's vitest config when present. Otherwise the shared `vitest.config.mjs` is used — an installed `@mayjournal/fitness-shared` package when present, else the copy embedded in the check binary, materialized on demand.
- When the project has no local vitest config, `vitest run --coverage` runs with `--config` pointing at that resolved file.
- The vitest binary resolves from `node_modules/.bin` (walking up from the project root), then PATH — never npx. A missing binary fails with a one-line install hint.

## Errors

```text
Vitest coverage thresholds must be 100 for branches, functions, lines, and statements.
Fitness-runner package must have Vitest coverage thresholds set to 100 for branches, functions, lines, and statements.
```

A failing coverage run reports the last line of vitest output, or a fallback hint when there is none:

```text
Vitest coverage did not meet 100% thresholds. Run: npm run test
```

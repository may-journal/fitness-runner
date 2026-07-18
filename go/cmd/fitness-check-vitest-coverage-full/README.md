---
relatedConfigurations: ['../../internal/sharedconf/config/vitest.config.mjs']
---

# vitest-coverage-full

## Behavior

- Pass: (1) The project where fitness runs has Vitest coverage thresholds set to 100 for branches, functions, lines, and statements; (2) The fitness-runner shared config has the same 100% thresholds; (3) `vitest run --coverage` in the project root exits 0.
- Fail: Thresholds not all 100 in the project → "Vitest coverage thresholds must be 100 for branches, functions, lines, and statements." Thresholds not all 100 in the fitness-runner shared config → "Fitness-runner package must have Vitest coverage thresholds set to 100 …". Run exits non-zero → last line of output or fallback.

Both the consumer project and the fitness-runner shared config are checked statically; neither may lower thresholds to pass. Threshold and exclude settings are read from the project's vitest config when present; otherwise from the shared `vitest.config.mjs` — an installed `@mayjournal/fitness-shared` package when present, else the copy embedded in the check binary, materialized on demand. When the project has no local vitest config, `vitest run --coverage` is invoked with `--config` pointing at that resolved file. The vitest binary itself is resolved from `node_modules/.bin` (walking up from the project root), then PATH — never npx; a missing binary fails with a one-line install hint.

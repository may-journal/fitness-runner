---
relatedConfigurations: ['../../../packages/shared/config/vitest.config.mjs']
---

# vitest-coverage-full

## Behavior

- Pass: (1) The project where fitness runs has Vitest coverage thresholds set to 100 for branches, functions, lines, and statements; (2) The @mayjournal/fitness package itself has the same 100% thresholds; (3) `vitest run --coverage` in the project root exits 0.
- Fail: Thresholds not all 100 in the project → `ThresholdsNot100`. Thresholds not all 100 in the @mayjournal/fitness package → `FitnessRunnerThresholdsNot100`. Run exits non-zero → last line of output or fallback.

Both the consumer project and the @mayjournal/fitness package are checked statically; neither may lower thresholds to pass. Threshold and exclude settings are read from the project's vitest config when present; otherwise from this package's `vitest.config.mjs`. When the project has no local vitest config, `vitest run --coverage` is invoked with `--config` pointing at that package export.

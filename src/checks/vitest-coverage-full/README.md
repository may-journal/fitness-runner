---
relatedConfigurations: ['vitest-coverage-exclude']
---

# vitest-coverage-full

## Behavior

- Pass: (1) The project where fitness runs has Vitest coverage thresholds set to 100 for branches, functions, lines, and statements; (2) The fitness-runner package itself has the same 100% thresholds; (3) `vitest run --coverage` in the project root exits 0.
- Fail: Thresholds not all 100 in the project → `ThresholdsNot100`. Thresholds not all 100 in the fitness-runner package → `FitnessRunnerThresholdsNot100`. Run exits non-zero → last line of output or fallback.

Both the consumer project and the fitness-runner package are checked statically; neither may lower thresholds to pass.
Runs the full test suite with coverage in the project root.

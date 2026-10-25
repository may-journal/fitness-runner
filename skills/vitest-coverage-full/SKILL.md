---
name: vitest-coverage-full
description: Fix a failing vitest-coverage-full check on Vitest coverage. Use when thresholds are below 100 or vitest run --coverage fails.
---

# Reach full Vitest coverage

Vitest coverage thresholds are 100 for branches, functions, lines and statements, and `vitest run --coverage` passes. [The check README](../../go/cmd/fitness-check-vitest-coverage-full/README.md) lists the three conditions.

## Read the failure

- `Vitest coverage thresholds must be 100 for branches, functions, lines, and statements.`: set each threshold in the project's vitest config back to 100.
- `Fitness-runner package must have Vitest coverage thresholds set to 100 for branches, functions, lines, and statements.`: the shared config was lowered. Restore 100 there.
- `Vitest coverage did not meet 100% thresholds. Run: npm run test`: run the tests and read the coverage table. Add tests for each uncovered line and branch, driving the real behavior.
- `vitest not found in node_modules/.bin (walking up from root) or on PATH. Install it (npm install -D vitest) to run the vitest-coverage-full check`: install vitest.

Never lower a threshold to pass.

## Check it first

Run `fitness vitest-coverage-full` from the repo root. Exit 0 means all thresholds are 100 and the run passes.

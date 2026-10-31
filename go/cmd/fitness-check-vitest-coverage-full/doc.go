// Command fitness-check-vitest-coverage-full requires Vitest coverage
// thresholds of 100 in the repo and the shared config, then runs
// `vitest run --coverage`.
//
// # Behavior
//
// A pass requires all three of:
//
//   - The project sets Vitest coverage thresholds to 100 for branches, functions, lines, and statements.
//   - The fitness-runner shared config carries the same 100% thresholds.
//   - `vitest run --coverage` exits 0 in the project root.
//
// Both configs are checked statically, as text. Neither side may lower
// thresholds to pass.
//
//   - Threshold and exclude settings come from the project's vitest config, else the shared `vitest.config.mjs` (installed `@mayjournal/fitness-shared` package or the binary's embedded copy).
//   - When the project has no local vitest config, `vitest run --coverage` runs with `--config` pointing at that resolved file.
//   - The vitest binary resolves from `node_modules/.bin` (walking up from root), then PATH — never npx. A missing binary fails with an install hint.
//
// # Errors
//
//	Vitest coverage thresholds must be 100 for branches, functions, lines, and statements.
//	Fitness-runner package must have Vitest coverage thresholds set to 100 for branches, functions, lines, and statements.
//
// A failing coverage run reports the last line of Vitest output when Vitest
// judged the code. It judged the code when a threshold is missed or a test
// failed. A run that exits non-zero with neither, could not start, or was
// killed reports info quoting its last lines. A missing coverage provider is
// one example.
//
//fitness:container cli
//fitness:component check Each check is its own package, built into the fitness binary.
package check

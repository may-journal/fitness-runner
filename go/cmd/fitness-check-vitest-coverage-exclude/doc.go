// Command fitness-check-vitest-coverage-exclude fails any Vitest coverage
// exclude entry, so coverage measures every file.
//
// # Behavior
//
//   - Pass: no Vitest config, or `coverage.exclude` is empty or missing.
//   - Fail: any exclude entry, tests and type files included → one error per entry, naming it to remove.
//   - Vitest's own default exclude skips tests and declarations, so set `coverage.exclude: []` to measure them; the shared config does.
//
// It reads `vitest.config.*` or the `package.json` `vitest` field in the
// project root. If none exist, it uses the shared vitest config. That is an
// installed `@mayjournal/fitness-shared` package when present, else the copy
// embedded in the check binary, materialized on demand.
//
// It reads the config as text, so exclude entries built from variables,
// spreads, or imports are invisible to it.
package check

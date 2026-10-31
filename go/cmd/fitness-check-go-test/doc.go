// Command fitness-check-go-test runs `go test ./...` in every Go module of
// the repo and reports the failing tests. It is on by default.
//
// # Behavior
//
//   - Finds each `go.mod` in the repo, leaving out `testdata` and `vendor` fixtures and paths the config ignores.
//   - Runs `go test ./...` in each module. A failure reports failing tests, panics, build errors, and each test's `file.go:line` messages.
//   - A repo with no `go.mod` passes clean (`filesChecked: 0`), so the check is safe in every repo.
//   - A run scoped to changed files skips when no `.go`, `go.mod`, or `go.sum` file changed.
//   - A missing Go toolchain reports info with a one-line install hint, never a failure.
//   - A `go test` that could not start, was killed, hit sandbox text, or failed without an explaining line reports info quoting its output.
//   - The time budget is 15 minutes, since a test suite can run long.
//
//fitness:container cli
//fitness:component check Each check is its own package, built into the fitness binary.
package check

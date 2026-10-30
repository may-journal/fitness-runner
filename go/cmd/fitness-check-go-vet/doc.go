// Command fitness-check-go-vet runs `go vet ./...` in every Go module of the
// repo, catching suspicious code the compiler accepts. It is on by default.
//
// # Behavior
//
//   - Finds each `go.mod` in the repo, leaving out `testdata` and `vendor` fixtures and paths the config ignores.
//   - Runs `go vet ./...` in each module and reports its findings, prefixed with the module directory when it is not the repo root.
//   - A repo with no `go.mod` passes clean (`filesChecked: 0`), so the check is safe in every repo.
//   - A run scoped to changed files skips when no `.go`, `go.mod`, or `go.sum` file changed.
//   - A missing Go toolchain reports info with a one-line install hint, never a failure.
//   - A `go vet` that could not start, was killed, hit sandbox text, or failed naming no `.go` file reports info quoting its output.
//   - `filesChecked` counts the modules vetted.
package check

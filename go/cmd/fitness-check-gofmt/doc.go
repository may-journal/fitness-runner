// Command fitness-check-gofmt lists Go files that gofmt would change. It is
// on by default.
//
// # Behavior
//
//   - Checks the repo's own `.go` files, leaving out `testdata` and `vendor` fixtures and paths the config ignores.
//   - Runs `gofmt -l` over them and reports each listed file as `path: not gofmt-formatted (run: gofmt -w path)`.
//   - A repo with no `.go` files passes clean (`filesChecked: 0`), so the check is safe in every repo.
//   - A file gofmt cannot parse fails as `path:line:column: message`.
//   - A missing Go toolchain reports info with a one-line install hint, never a failure.
//   - A gofmt that could not start, was killed, or exits non-zero with neither finding reports info quoting its output.
//   - `filesChecked` counts the Go files checked.
//
//fitness:container cli
//fitness:component check Each check is its own package, built into the fitness binary.
package check

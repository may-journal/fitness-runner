---
relatedConfigurations: ['../../../.fitnessrc.json']
---

# go-test

Runs `go test ./...` in every Go module of the repo and reports the failing tests. On by default.

## Behavior

- Finds each `go.mod` in the repo, leaving out `testdata` and `vendor` fixtures and paths the config ignores.
- Runs `go test ./...` in each module. A failure reports failing tests, panics, build errors, and each test's `file.go:line` messages.
- A repo with no `go.mod` passes clean (`filesChecked: 0`), so the check is safe in every repo.
- A run scoped to changed files skips when no `.go`, `go.mod`, or `go.sum` file changed.
- A missing Go toolchain fails with a one-line install hint.
- The time budget is 15 minutes, since a test suite can run long.

---
name: go-test
description: Fix a failing go-test check on a Go module. Use when go test reports failing tests, panics or build errors.
---

# Make the Go tests pass

Every Go module in the repo passes `go test ./...`. [The check README](../../go/cmd/fitness-check-go-test/README.md) says how modules are found and when the check skips.

## Read the failure

- `--- FAIL`, `FAIL`, `panic:` and `file.go:line` lines: each names a failing test, package, panic, build error or test message.
- A module outside the repo root prefixes its lines with the module directory.
- Fix the production code the test exercises. Change the test only when its expectation is wrong.
- `go test failed (run: go test ./...)`: no line explained the failure. Run that command in the module and read the output.
- `Go not installed: see https://go.dev/dl`: install Go, then rerun.

The check skips a scoped run when no `.go`, `go.mod` or `go.sum` file changed. It allows 15 minutes, so fix a hanging test.

## Check it first

Run `fitness go-test` from the repo root. Exit 0 means it passes.

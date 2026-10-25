---
name: go-vet
description: Fix a failing go-vet check on a Go module. Use when go vet reports suspicious code.
---

# Fix go vet findings

Every Go module passes `go vet ./...`. [The check README](../../go/cmd/fitness-check-go-vet/README.md) says how modules are found and when the check skips.

## Read the failure

- `file.go:line:col: message`: a vet finding. Fix the code it points at, such as a wrong format verb or a copied mutex.
- A module outside the repo root prefixes its lines with the module directory.
- `go vet failed (run: go vet ./...)`: vet failed without a finding line. Run the command in that module and read its output.
- A missing Go toolchain fails with an install hint. Install Go, then rerun.

The check skips a scoped run when no `.go`, `go.mod` or `go.sum` file changed.

## Check it first

Run `fitness go-vet` from the repo root. Exit 0 means it passes.

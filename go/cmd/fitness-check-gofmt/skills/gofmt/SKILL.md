---
name: gofmt
description: Fix a failing gofmt check on Go files. Use when a Go file is reported as not gofmt-formatted.
---

# Format Go files with gofmt

Every `.go` file matches `gofmt` output. [The check README](../../README.md) lists which files are checked.

## Read the failure

- `go/cmd/tool/main.go: not gofmt-formatted (run: gofmt -w go/cmd/tool/main.go)`: run the command on that path, then review the diff and commit it. Each listed file gets one line.
- A missing Go toolchain fails with an install hint. Install Go, then rerun.

Do not edit by hand to match. Let `gofmt -w` rewrite the file.

## Check it first

Run `fitness gofmt` from the repo root. Exit 0 means no file would change.

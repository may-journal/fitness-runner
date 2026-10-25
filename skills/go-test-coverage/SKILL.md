---
name: go-test-coverage
description: Fix a failing go-test-coverage check on Go entry packages. Use when an entry is below 100% statement coverage or a test wraps a helper in t.Run.
---

# Cover every entry statement

Every command package and declared entry package reaches 100% statement coverage through its own tests. [The check README](../../go/cmd/fitness-check-go-test-coverage/README.md) explains entries, the overlap audit and the source map.

## Read the failure

- `entry example.com/team/project/cmd/tool: 9/10 statements; required 100%`: one statement never ran. Add a test that drives the entry to it.
- `test wrapper: file_test.go:12: TestX passes named helper h to t.Run; ...`: write each case inline in its own `t.Run` call, or delete the duplicate test.

Child packages earn coverage from entry tests, so test behavior through the entry.
- `unknown entry package "<name>"`: correct the import path in `goTestCoverage.entries` of `.fitnessrc.json`.

Exact counts decide, so 99.99% still fails. Tests that launch a Go binary build it with `go build -cover` and set the child's `GOCOVERDIR` to `FITNESS_GO_COVER_DIR`.

## Check it first

Run `fitness go-test-coverage --all` from the repo root. Exit 0 means every entry is at 100%.

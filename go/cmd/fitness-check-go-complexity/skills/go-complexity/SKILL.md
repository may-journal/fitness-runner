---
name: go-complexity
description: Fix a failing go-complexity check on Go functions. Use when a function is reported as too complex.
---

# Lower a Go function's complexity

Each Go function stays at or under the complexity ceiling, 5 by default. [The check README](../../README.md) lists how branches are counted.

## Read the failure

- `go/internal/example/thing.go:42: func parseAll has a complexity of 8; maximum allowed is 5`: split it into smaller functions that each do one step, or replace branches with a table lookup.
- A function starts at 1 and gains a point per `if`, `for`, `range`, non-default `case`, `&&` and `||`.
- Function literals report as `function literal` and methods as `method <name>`. A literal scores apart from the function around it.
- A parse error means the file does not compile. Fix the syntax first.

Test files count too. Write table-driven tests so each test function stays small.

Fix the function. Do not raise `goComplexity.max` in `.fitnessrc.json`.

## Check it first

Run `fitness go-complexity` from the repo root. Exit 0 means it passes.

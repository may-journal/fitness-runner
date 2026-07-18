---
fitnessFunctions: ['main.go']
relatedConfigurations: ['../../../.fitnessrc.json']
---

# go-complexity

Enforces a cyclomatic complexity ceiling on Go functions — the Go-native counterpart of the house eslint rule (`complexity: max 5`) that gates JavaScript and TypeScript.

## Behavior

- Every function starts at 1 and gains a point per branch: `if`, `for`, `range`, each non-default `switch` case or `select` clause, and each `&&` or `||`.
- Function literals score separately from their enclosing function, the way eslint scores arrow functions.
- Test files (`_test.go`) are exempt, matching the repo's test-file exemptions elsewhere.
- A file that fails to parse fails the check with the parser's error.
- `filesChecked` counts the non-test Go files scanned.

## Errors

```text
go/internal/example/thing.go:42: func parseAll has a complexity of 8; maximum allowed is 5
```

Function literals report as `function literal`; methods report as `method <name>`.

## Configuration

The ceiling defaults to 5. Override per repo in `.fitnessrc.json`:

```json
{
  "goComplexity": { "max": 8 }
}
```

## Enable

Opt-in (Go-specific, like `swiftlint`) — add the name to your `checks` list:

```json
{
  "checks": ["go-complexity"]
}
```

Native Go engine (`go/ast` + `go/parser` from the standard library); no external tool required.

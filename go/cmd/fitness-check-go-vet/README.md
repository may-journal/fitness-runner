---
relatedConfigurations: ['../../../.fitnessrc.json']
---

# go-vet

Runs `go vet ./...` in every Go module of the repo, catching suspicious code the compiler accepts. On by default.

## Behavior

- Finds each `go.mod` in the repo, leaving out `testdata` and `vendor` fixtures and paths the config ignores.
- Runs `go vet ./...` in each module and reports its findings, prefixed with the module directory when it is not the repo root.
- A repo with no `go.mod` passes clean (`filesChecked: 0`), so the check is safe in every repo.
- A run scoped to changed files skips when no `.go`, `go.mod`, or `go.sum` file changed.
- A missing Go toolchain fails with a one-line install hint.
- `filesChecked` counts the modules vetted.

## Turn off

```json
{ "disabledChecks": ["go-vet"] }
```

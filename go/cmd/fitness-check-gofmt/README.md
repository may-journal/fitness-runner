---
relatedConfigurations: ['../../../.fitnessrc.json']
---

# gofmt

Lists Go files that `gofmt` would change. On by default.

## Behavior

- Checks the repo's own `.go` files, leaving out `testdata` and `vendor` fixtures and paths the config ignores.
- Runs `gofmt -l` over them and reports each listed file as `path: not gofmt-formatted (run: gofmt -w path)`.
- A repo with no `.go` files passes clean (`filesChecked: 0`), so the check is safe in every repo.
- A missing Go toolchain fails with a one-line install hint.
- `filesChecked` counts the Go files checked.

## Turn off

```json
{ "disabledChecks": ["gofmt"] }
```

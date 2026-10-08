---
relatedConfigurations: ['../.fitnessrc.json']
---

# Check protocol

The runner invokes each check as `fitness-check-<name> --root <dir> [args…]` with cwd set to the repo root. Context arrives in `FITNESS_*` environment variables: staged files, changed files, enabled check names, and the commit message for the commit checks.

- stdout — one JSON result object: `{"ok": bool, "errors": [".."], "filesChecked": n}`
- stderr — human display output (banners, tool passthrough)
- exit code — 0 when the check ran and passed, 1 ran and failed, other values mean it crashed
- `--describe` — prints check metadata (name, timeout budget, context-inline arg) so the runner needs no registry

The runner executes checks in a bounded parallel pool with per-check timeouts. A timeout kills the whole process group, so a hung check's child tree dies with it. Results render as a summary table, and the run exits 1 when any check fails.

## Applicability

Org mode runs every check, so each owes the runner one more rule. In a repo it does not apply to, it passes with `filesChecked: 0`. The contract test in `go/cmd/fitness/contract_test.go` builds every check and holds each to this and to a matching `--describe` name. The few checks that apply everywhere, such as `semantic-commit`, are listed there with a reason.

## Changed files

A scoped run lists changed files in `FITNESS_CHANGED_FILES`: the staged files locally, or a pull request's diff against its base. A push to `main` or `fitness --all` leaves it unset, so every file is checked. Checks that compare files, such as jscpd, keep a full walk.

## Config

Optional `.fitnessrc.json` at repo root:

```json
{
  "repeatedStringLiterals": { "allow": ["dist"] }
}
```

Org mode runs every check; each passes clean when it does not apply. `timeoutMs` replaces the 5-second default budget for checks that declare none. A leftover `checks` list is ignored with a warning in org mode.

External mode sets `"policy": "external"` and runs all checks by default; a nonempty `"checks"` list is an optional filter. `--policy=external --checks=name,name` can set both on the CLI; CLI selection replaces the entire config list. Unknown, empty, and duplicate names fail. See [external CI examples](ci.md) for complete setup.

## Every tracked file

File checks judge every tracked file through `walkfs`; outside git, everything but `.git`. A config setting `ignore`, `skipTheseDirectories`, `disabledChecks`, or `proseBudget.exempt` fails.


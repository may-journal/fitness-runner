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

Every check runs in every repo, so each owes the runner one more rule. In a repo it does not apply to, it passes with `filesChecked: 0`. The contract test in `go/cmd/fitness/contract_test.go` builds every check and holds each to this and to a matching `--describe` name. The few checks that apply everywhere, such as `semantic-commit`, are listed there with a reason.

## Changed files

A scoped run lists changed files in `FITNESS_CHANGED_FILES`: the staged files locally, or a pull request's diff against its base. A push to `main` or `fitness --all` leaves it unset, so every file is checked. Checks that compare files, such as jscpd, keep a full walk.

## Config

Optional `.fitnessrc.json` at repo root:

```json
{
  "disabledChecks": ["cspell"],
  "ignore": ["profile/README.md", ".github/workflow-templates/**"],
  "repeatedStringLiterals": { "allow": ["dist"] }
}
```

Every check runs in every repo, and each passes clean when its language, tool, config, or input is absent. `disabledChecks` turns one off by name. A leftover `checks` list is ignored with a warning.

## Ignored paths

`ignore` lists paths every file check skips, with gitignore-like globs. A bare name matches at any depth, `**` crosses directories, and a matched directory hides everything beneath it.


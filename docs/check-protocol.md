---
relatedConfigurations: ['../.fitnessrc.json']
---

# Check protocol

The runner invokes each check as `fitness-check-<name> --root <dir> [args…]` with cwd set to the repo root. Context arrives in `FITNESS_*` environment variables: staged files, enabled check names, and the commit message for the commit checks.

- stdout — one JSON result object: `{"ok": bool, "errors": [".."], "filesChecked": n}`
- stderr — human display output (banners, tool passthrough)
- exit code — 0 when the check ran and passed, 1 ran and failed, other values mean it crashed
- `--describe` — prints check metadata (name, timeout budget, context-inline arg) so the runner needs no registry

The runner executes checks in a bounded parallel pool with per-check timeouts. A timeout kills the whole process group, so a hung check's child tree dies with it. Results render as a summary table, and the run exits 1 when any check fails.

## Config

Optional `.fitnessrc.json` at repo root:

```json
{
  "disabledChecks": ["cspell"],
  "enableChecks": ["commit-attribution"],
  "ignore": ["profile/README.md", ".github/workflow-templates/**"],
  "repeatedStringLiterals": { "allow": ["dist"] }
}
```

Most repos set no check list. The default list holds every check that judges repo files, and each passes clean when its language, tool, or file is absent. `disabledChecks` removes names; `enableChecks` appends opt-in checks such as `commit-attribution`.

## Ignored paths

`ignore` lists paths every file check skips, with gitignore-like globs. A bare name matches at any depth, `**` crosses directories, and a matched directory hides everything beneath it.

## Check lists

A `checks` list replaces the defaults: only those run, in order, and `enableChecks` is ignored. Unknown names are skipped silently, and `disabledChecks` never removes path entries.

Entries containing `/` are local executable paths, mixed in with check names. This runs a repo-specific check without publishing anything. A local check is any executable speaking the protocol above — a shell script works.

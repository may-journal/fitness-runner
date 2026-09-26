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
  "checks": ["changelog", "node-version", "semantic-commit"],
  "disabledChecks": ["cspell"],
  "repeatedStringLiterals": { "allow": ["dist"] }
}
```

If `checks` is set, only those run (in order). If omitted, the runner uses its default list. `disabledChecks` removes names from either list. Unknown names in `checks` are skipped silently; `disabledChecks` never removes path entries.

`checks` entries can also be local executable paths — entries containing `/` — mixed in with check names. This runs a repo-specific check without publishing anything. A local check is any executable speaking the protocol above — a shell script works.

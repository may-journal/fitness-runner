---
relatedConfigurations: ['../package.json']
issue: https://github.com/may-journal/fitness-runner/issues/23
---

# Plan: Local check paths in `.fitnessrc` (#23)

Let `.fitnessrc` `checks` mix npm names and local paths — same as CLI `--check=./foo.js` already works today.

Architecture: [architecture/03-components.md](../architecture/03-components.md#resolve-check-specs-priority)

## Already done

- CLI can run a check from a path
- Path checks run in-process
- C4 docs describe the target behavior

## Build

1. Move path loading (`isPathSpec`, `loadCheckFromPath`) from `run-resolve.ts` → `load-check.ts` — CLI path checks still work; one code path for CLI and config.

2. When resolving config `checks`, branch on spec kind: name → `loadCheck`; path → `loadCheckFromPath` (fail if missing/invalid) — `checks: ['cspell', './fitness/my-check.js']` runs both in order.

3. `disabledChecks` still only filters names — path entries in `checks` are never removed by `disabledChecks`.

4. Widen `FitnessConfig.checks` to accept paths — types match runtime behavior.

5. Tests + README example — mixed config, bad path error, docs show a local check module.

## Not doing

- Globs, `.ts` without build, `disabledChecks` for paths

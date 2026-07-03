---
relatedConfigurations: ['../package.json']
issue: https://github.com/may-journal/fitness-runner/issues/23
---

# Plan: Local check paths in `.fitnessrc` (#23)

> Let `.fitnessrc` `checks` mix npm names and local paths — same as CLI `--check=./foo.js` already works today.

## Goal

A repo can list a local check module path in `.fitnessrc` `checks` alongside npm check names, in order, without publishing an `@mayjournal/fitness-check-*` package for one-off rules.

Architecture: [03-components.md](../architecture/03-components.md#resolve-check-specs-priority) already documents this as the target behavior.

## Plan

0. Already working (CLI single-check mode)
   - [x] `npx fitness --check=./foo.js` loads a path spec (`isPathSpec` / `loadCheckFromPath` in `run-resolve.ts`)
   - [x] Path-loaded checks run in-process, not in a worker (`markPathLoadedCheck`)

1. Share path loading between CLI and config
   - [ ] Move `isPathSpec` / `loadCheckFromPath` from `run-resolve.ts` to `load-check.ts` — one code path for both callers

2. Support paths in config `checks`
   - [ ] `resolveBaseCheckNames` (`load-check.ts`) branches per entry: name → `loadCheck`, path → `loadCheckFromPath` (fail loud if missing/invalid — it was explicitly configured)
   - [ ] `checks: ['cspell', './fitness/my-check.js']` runs both, in order

3. `disabledChecks` stays name-only
   - [ ] Path entries in `checks` are never removed by `disabledChecks` (opt-in only, same as today's design intent)

4. Types
   - [ ] Widen `FitnessConfig.checks` to accept paths so types match runtime behavior

5. Tests + docs
   - [ ] Test: mixed name/path config runs both
   - [ ] Test: bad/missing path in config throws
   - [ ] README example showing a local check module referenced by path

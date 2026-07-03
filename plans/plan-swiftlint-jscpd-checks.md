---
relatedConfigurations: ['../package.json']
---

# Plan: SwiftLint and jscpd checks

> Add `swiftlint` and `jscpd` as opt-in checks, matching what bottom-line/may-journals already run outside fitness.

## Goal

bottom-line and may-journals install `@mayjournal/fitness` today but run SwiftLint (`make swift-lint`, manual) and jscpd (`npm run duplication`, chained into `precommit`) separately. Two new opt-in check packages fold both into the same `checks:` list and results table. Neither joins `defaultChecks` — pure JS/TS consumers don't need a Swift binary or a duplication scan by default.

## Plan

0. Shared groundwork
   - [ ] Model both on `packages/checks/cspell/src/runCspell.ts` — `execSyncResult` + `buildExecCheckResult`, `_execSync` context override for tests (no real binaries in this repo's CI)
   - [ ] Add both names to `packages/checks-bundle/scripts/bundle-check-dist.mjs` `CHECK_NAMES` (not `defaultChecks`) — that array is what actually gets a check into the published bundle

1. `jscpd` check — `packages/checks/jscpd/`
   - [ ] Default invocation: `jscpd --gitignore --min-lines 5 --min-tokens 50 --threshold 1 --reporters console .` — proven thresholds from bottom-line/may-journals; dropped their repo-specific `--format swift` and path list so one check works for any repo
   - [ ] Add `jscpd` as a real dependency on `@mayjournal/fitness-shared` (same place `prettier`/`vitest` already live) so it actually installs for consumers — `@mayjournal/fitness-checks` ships zero dependencies today
   - [ ] Tests mock `_execSync` for pass / fail / threshold-exceeded
   - [ ] README: flags, threshold, `jscpd:ignore-start/end` escape hatch

2. `swiftlint` check — `packages/checks/swiftlint/`
   - [ ] Shells out to the `swiftlint` binary (`lint --strict --reporter json`) — no npm dependency, no library fallback
   - [ ] Missing binary → clear `ok: false` error, not a thrown exception; no Swift files → skip clean
   - [ ] Tests mock `_execSync` for pass / fail / missing-binary / no-swift-files
   - [ ] README: binary prerequisite, `--strict` meaning

3. Docs
   - [ ] README example `.fitnessrc` enabling both
   - [ ] `architecture/02-containers.md` / `03-components.md` note them as opt-in checks with an external dependency

4. Verification
   - [ ] `npm run fitness` + `npm run lint` green here (binaries mocked, not real)
   - [ ] Dry run in bottom-line: add `jscpd`/`swiftlint` to `checks`, drop the `duplication` script and the SwiftLint half of `make swift-check` from `precommit`

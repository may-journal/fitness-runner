---
relatedConfigurations: ['../../../.fitnessrc.json']
---

# Plan: SwiftLint and jscpd checks

> Add `swiftlint` and `jscpd` checks, matching what bottom-line/may-journals already run outside fitness.

## Goal

bottom-line and may-journals install `@mayjournal/fitness` today but run SwiftLint (`make swift-lint`, manual) and jscpd (`npm run duplication`, chained into `precommit`) separately. Two new check packages fold both into the same `checks:` list and results table.

`jscpd` joins `defaultChecks` — duplicate-code detection is broadly useful, not Swift-specific. `swiftlint` stays opt-in only — most repos have no Swift code and no `swiftlint` binary.

Adding `jscpd` to `defaultChecks` means it also runs on this repo: at 2.3% duplication (mostly structural boilerplate across ~12 near-identical check packages), it fails the proven 1% threshold. Rather than force a refactor, this repo's own `.fitnessrc.js` disables it with a comment explaining why — `jscpd` still ships as a real default for every other consumer.

## Plan

0. Shared groundwork
   - [x] Model both on `packages/checks/cspell/src/runCspell.ts` — `execSyncResult` + `buildExecCheckResult`, `_execSync` context override for tests (no real binaries in this repo's CI)
   - [x] Add both names to `packages/checks-bundle/scripts/bundle-check-dist.mjs` `CHECK_NAMES` — that array is what actually gets a check into the published bundle, independent of `defaultChecks`
   - [x] `jscpd` joins `packages/checks-bundle/src/index.ts` `defaultChecks`; `CheckName` enum stays exactly in sync with `defaultChecks` (`registry.test.ts` enforces this) — `swiftlint` is deliberately not in the enum, since it never joins `defaultChecks`

1. `jscpd` check — `packages/checks/jscpd/`
   - [x] Default invocation: `jscpd --min-lines 5 --min-tokens 50 --threshold 1 --ignore "**/*.md,**/*.json,**/*.lock" --reporters console .` — proven thresholds from bottom-line/may-journals; dropped their repo-specific `--format swift` and path list so one check works for any repo. Ignoring markdown/JSON/lockfiles was necessary generally, not just for this repo — without it jscpd flags structurally-similar `package.json`/doc scaffolding as "duplication," which isn't real code to refactor
   - [x] Add `jscpd` as a real dependency on `@mayjournal/fitness-shared` (same place `prettier`/`vitest` already live) so it actually installs for consumers — `@mayjournal/fitness-checks` ships zero dependencies today
   - [x] Tests mock `_execSync` for pass / fail / threshold-exceeded (7 tests, 100% coverage); verified against the real `jscpd` binary too
   - [x] README: flags, threshold, `jscpd:ignore-start/end` escape hatch
   - [x] This repo's own `.fitnessrc.js` disables `jscpd` (2.3% duplication here, over threshold — structural, not worth a forced refactor right now)

2. `swiftlint` check — `packages/checks/swiftlint/`
   - [x] Shells out to the `swiftlint` binary (`lint --strict --reporter json --quiet`) — no npm dependency, no library fallback
   - [x] Missing binary → clear `ok: false` error, not a thrown exception; no Swift files → skip clean (both verified against the real `swiftlint` binary)
   - [x] Tests mock `_execSync` for pass / fail / missing-binary / no-swift-files (8 tests, 100% coverage)
   - [x] README: binary prerequisite, `--strict` meaning

3. Docs
   - [x] README: `jscpd` documented under defaults, `swiftlint` under an opt-in section with an example
   - [x] `architecture/02-containers.md` / `03-components.md` note `swiftlint` as the one opt-in, no-npm-dependency check

4. Verification
   - [x] `npm run fitness` + `npm run lint` green here (binaries mocked in unit tests; both also verified against real installed binaries)

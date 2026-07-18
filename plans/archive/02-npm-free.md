---
relatedConfigurations: ['../../.fitnessrc.json']
status: completed
completedAt: 2026-07-18
---

# 02 — npm-free: nothing left but Go

> The product is pure Go but the repo still drags `node_modules`, package.json, and node scripts behind it. Remove every last npm anchor — embedded configs, Go hook tooling, frozen dictionaries.

## Goal

`git clone && cd go && go build -o bin ./cmd/...` is the entire toolchain. No package.json, no `node_modules`, no npm scripts, no node in the hooks. Consumers lose nothing: the shared tool configs ship inside the check binaries and materialize on demand when a repo has no local config, existing `node_modules/@mayjournal/fitness-shared` installs keep winning over the embedded fallback, and every check stays in the catalog — only this repo's dogfood list drops the checks that need tools npm used to install here.

## Plan

0. Configs move into the binaries

   - [x] `internal/sharedconf` embeds the shared config directory (eslint.base/eslint.config/prettier.config/cspell.json/constants/vitest.config) and materializes it to a content-keyed cache dir on demand
   - [x] The five fallback consumers (eslint, prettier, cspell, vitest-coverage-exclude, vitest-coverage-full) resolve local config, then an installed `@mayjournal/fitness-shared`, then the embedded copy
     - [x] The `@mayjournal/fitness-shared` npm package retires; `packages/` is deleted (published versions stay on npm untouched)

1. Checks learn to live without package.json

   - [x] `changelog` treats an absent package.json as "no version rules to enforce" (heading format still gated) — a correction for non-npm repos generally, not a special case for this one
   - [x] `dependency-currency`, `eslint`, `prettier`, and `node-version` leave this repo's `.fitnessrc.json` (nothing for them to judge here anymore); all stay in the catalog for consumer repos

2. The dev loop goes Go

   - [x] `fitness-stamp-changelog` binary replaces `scripts/ensure-changelog-timestamp` in pre-commit: restamps the first heading when CHANGELOG.md is staged, re-stages, no version bumps, no npm install
   - [x] githooks call `go build` + the binaries directly; nvm-use and every node script under `scripts/` is deleted along with the npm publish workflows and the node CI setup action
   - [x] The spell dictionaries freeze as committed data; `generate.mjs` and the cspell devDependencies go — a Go regeneration tool that fetches dictionary sources over HTTPS is deferred to a later milestone

3. Delete the anchors

   - [x] package.json, package-lock.json, `node_modules`, .npmrc, .nvmrc removed; .gitignore keeps only entries that still earn their why-comment
   - [x] README and the architecture docs describe the Go-only toolchain (clone, build, PATH — no npm anywhere)

4. Lock it in

   - [x] `go vet` + `go test ./...` green; the trimmed dogfood suite passes on this repo
   - [x] CI reduced to Go jobs only and the fitness job runs without node setup
   - [x] A real commit lands through the rewritten hooks end to end (stamp, suite, commit-msg validation)

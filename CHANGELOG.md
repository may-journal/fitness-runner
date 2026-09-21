---
# Changed package version should correlate with this file
relatedConfigurations: ['.fitnessrc.json']
---

# Changelog

## Changes

### 2026.09.20.1941

- Docs: add a `Plan` issue template (`.github/ISSUE_TEMPLATE/plan.md`) modeled on the personal `me` repo's — a one-line pitch, `## Background`, a `## What needs to happen` checklist, and no other sections.
- Chore: migrate every `docs/plans` file into a GitHub Issue under the new `Plan` label — the two active plans stay open (#49, #50) and the eight archived plans become closed completed records (#51 through #58).
- Chore: delete the migrated `docs/plans` tree now that plans live as Issues, moving the `README.md` and `architecture-index.md` links from the plan files to their issues.
- Docs: open #59 to run fitness checks as GitHub workflows, so a plan Issue is validated by a workflow comment the way a checked-in plan file used to be validated by the runner.

### 2026.09.20.1417

- Feat: exempt capitalized doc basenames (README.md, LICENSE.md, AGENTS.md, CODE_OF_CONDUCT.md, ...) from the kebab-case and camelCase filename checks, so conventional all-caps docs pass at any path.
- Refactor: replace the hardcoded `allowedBasenames` map with a single all-caps basename pattern, dropping the fixed OSS-doc list in favor of one rule.
- Test: cover the all-caps exemption — README and AGENTS pass in root and nested paths, while lowercase and mixed-case names stay held to their convention.

### 2026.08.25.1516

- Chore: stop tracking `go/internal/spell/.DS_Store`, a macOS Finder metadata file that a `git add -A` swept into the previous commit.
- Chore: add a `.DS_Store` rule to `.gitignore` as a bare name, excluding the artifact from every directory rather than only the repo root.
- Chore: annotate that rule with a why-comment (Finder view-state metadata, no project value) so it satisfies the `gitignore-why` check.

### 2026.08.25.1511

- Chore: finish the npm-free migration — delete the now-unused Node `generate.mjs` generator (the 14 committed wordlists are the sole source, their headers and two Go comments rewritten to match) and remove a dead `.env` npm token plus its orphaned `.gitignore` allow rule.
- Fix: the embedded fallback tool configs targeted the retired `packages/runner/src` layout — vitest coverage globs now use the conventional `src` tree with an index-file exclude, eslint drops a dead `packages/shared/types` ignore, and a stale `constants.cjs` comment goes.
- Chore: tailor the check suite for this Go repo — disable the JavaScript-only checks via a new `disabledChecks` list in `.fitnessrc.json`, drop eslint, prettier, node-version, and the two vitest checks from the built-in `defaultChecks`, and add a root `Makefile` wired into the pre-commit hook.
- Fix: the cspell check now honors `ignorePaths` for staged files, not only the markdown walk, so committing the dictionary sources under `go/internal/spell/dict` (already listed in `ignorePaths`) no longer fails on the fragments inside them.
- Docs: add ADR 0002 retiring 0001's dual `fitness-shared lint` path, a pointer README for the camelCase filename check, a Cursor rule now aimed at `go/cmd`, and fixes to the README suite count and a moved plan-doc link.

### 2026.07.19.0934

- Feat: add the `markdown-links` check — every relative link in every markdown file must resolve to a real file or directory; absolute URLs are never touched, so the check stays offline and deterministic. The catalog is 31 names; the dogfood suite is 21 checks.
- Fix: the new check found nine broken links on arrival — two path-depth bugs from the docs consolidation (competition.md and an archived plan's changelog links) plus six archived-plan links to files the npm purge deleted, now honest code spans noting the removal.
- Docs: check counts across README and the architecture docs move to 31; the check's README documents the fence and code-span masking and the fragment-stripping rule.

### 2026.07.19.0915

- Docs: architecture docs catch up with the last two days — check counts move from 27 to 30 across system context, containers, and code levels, and the monorepo layout tree gains `.github/`, `docs/`, the stamper binary, and `internal/par`.
- Fix: the containers doc's README link had pointed at `docs/README.md` since the docs consolidation — now `../../README.md`, with a sentence on artifact shipping via Releases and the Go module proxy.
- Docs: the architecture index links the research directory and plan 03 alongside the two archived milestone plans.

### 2026.07.19.0909

- Docs: README install instructions now lead with the working channels — `go install` with pin and upgrade commands, then prebuilt release tarballs with a verified download URL, platform list, and checksum note; build-from-source moves to a contributors block.
- Docs: the Distribution section drops the pre-launch phrasing — the first two channels are live, and the Homebrew item now points at issue #47 as planned work.
- Chore: the documented release download URL was tested against the published release before landing (both the plain and encoded tag forms serve the asset).

### 2026.07.19.0859

- Feat: the repo is public — `go install github.com/may-journal/fitness-runner/go/cmd/...` now resolves through the public Go module proxy; verified from a clean environment with all 32 binaries installed and running.
- Chore: the Homebrew tap is deferred to issue #47; the release-side automation for it is already in place and skips politely until the tap exists.
- Docs: plan 03 flips the public-flip and go-install verification boxes — only the tap items remain open, each annotated with the issue.

### 2026.07.19.0852

- Fix: release asset names carried a stray leading v (the workflow stripped only the tag's directory prefix), so the tap script could never match its checksums — the workflow now strips the full prefix and asset names agree with the formula generator.
- Chore: first release published end to end from the tag the changelog derived — both workflow jobs green, five assets, and the downloaded darwin binary runs on this machine.
- Docs: plan 03 pre-flight and release-automation boxes flip; the public flip and tap creation remain open.

### 2026.07.19.0844

- Fix: release versions now derive from the changelog heading instead of hand-cut semver — heading timestamp `2026.07.19.0837` maps to tag `go/v0.20260719.837`; major pinned at 0 because Go reserves higher majors for `/vN` module paths, and date and minute keep their ordering.
- Feat: add `.github/scripts/release-tag.sh` — prints the tag for the newest heading; the release workflow refuses any tag that does not match it.
- Docs: README and plan 03 state the one-version rule — the pre-commit stamper owns the version, releases only transcribe it.

### 2026.07.19.0837

- Feat: add the release workflow — every `go/vX.Y.Z` tag cross-compiles static tarballs for darwin and linux on both architectures, writes a checksums file, and publishes a GitHub Release with the newest CHANGELOG section as its notes.
- Feat: add `.github/scripts/update-tap.sh` — regenerates the Homebrew formula from the release checksums and pushes it to `may-journal/homebrew-tap`; the workflow job skips politely until a `TAP_PUSH_TOKEN` secret exists.
- Docs: add `docs/plans/03-publish.md` — the publishing milestone in the may-journals template: pre-flight history audit, release automation, brew tap, verification last.

### 2026.07.19.0821

- Docs: README gains a Distribution section — the three decided channels layered on one artifact host: `go install` via the Go module proxy, GitHub Releases with prebuilt per-platform tarballs, and a Homebrew tap whose formula the release workflow bumps.
- Docs: the upgrade story is stated per channel — re-run with `@latest`, grab the next release, or `brew upgrade`.
- Docs: versioning documented — the CHANGELOG timestamp stays the internal version; releases are semver tags in the `go/vX.Y.Z` subdirectory-module form, referenced by consumers as plain `@vX.Y.Z`.

### 2026.07.19.0812

- Feat: add the `text-readability` check — a document-level readability smoke detector that scores every markdown file with the three character-based formulas and fails only when 2 of 3 exceed their alarm band; files under 100 prose words are never judged. The catalog grows to 30 names.
- Feat: failure output is structured as prompt fuel for an LLM — per-file alarms show each formula's value and band, then guidance lines carry the exact formulas, the edits that lower them, and pointers to the check README and the research doc.
- Chore: enable it here at tightened bands (grade 15, LIX 50) via the new `textReadability` config; shipped defaults stay at the calibrated smoke-detector bands (18/60). The dogfood suite is 20 checks.
- Docs: fix the three files the tightened bands flagged — README.md, the research doc, and the vitest-coverage-full README — by splitting long sentences, wrapping check names in code spans, and moving quoted tool output into fenced blocks; stale README counts corrected in passing.

### 2026.07.18.1940

- Docs: the prose-complexity research now uses the archived plan documents as its experiment corpus — the changelog was a weak example (bullets are notation, and its one clean separator was true by construction).
- Docs: the plans result is stronger and inverted — every readability formula scores the preferred may-journals-template plans as harder than the rejected free-form plans, and one uniformly written plan swings 17 grade levels paragraph to paragraph.
- Docs: the changelog numbers stay only as corroboration; the implications section now names both house interventions (`changelog-bullets`, the plan template) as structural gates that beat formulas.

### 2026.07.18.1931

- Docs: rename the research doc to `docs/research/0001-prose-cognitive-complexity.md`.
- Docs: research docs now carry a `NNNN-` number prefix, matching the ADR convention (`docs/architecture/adr/0001-...`), so they order by arrival.
- Chore: a pure `git mv` — content, front matter, and relative paths are unchanged.

### 2026.07.18.1930

- Docs: add `docs/research/prose-cognitive-complexity.md` — whether a check can judge the cognitive complexity of written paragraphs: the classical readability formulas, the cognitive-science measures beyond them, and prior CI tooling, surveyed with sources.
- Docs: the research includes an experiment on this repo's own changelog rewrite — no classical formula separates tight bullets from essay bullets one bullet at a time; only whole-section scoring and the existing 365-character cap discriminate cleanly.
- Docs: verdict for a future check — structural budgets and character-based formulas at document scale behind a frozen code-masking spec; sentence-connection measures are the unexplored ground.

### 2026.07.18.1909

- Feat: `changelog-bullets` now judges every `###` section of CHANGELOG.md, not only the newest — count errors name their section.
- Docs: rewrite the entire changelog history into compliance (219 findings to zero) — every heading byte-identical, facts and issue refs preserved, essays split into typed bullets, thin sections filled from their commits' real diffs.
- Docs: the check README now documents whole-file semantics.
- Test: `TestEverySectionIsJudged` pins the new scope; doc fixtures gain a compliant trailing section.

### 2026.07.18.1848

- Feat: add the `changelog-bullets` check — the newest CHANGELOG section must be 3-5 bullets, each under 365 characters, each with a semantic type prefix.
- Chore: enable it in this repo's dogfood list (19 checks) and add it to the catalog (29 names).
- Docs: rewrite the previous entry to comply — it was one 1,230-character bullet, precisely the style this check exists to end.

### 2026.07.18.1844

- Perf: cspell lookups now binary-search the embedded sorted dictionary bytes with a `sync.Map` memo — zero startup parsing, 7.2x faster on small repos, ~30ms on this one.
- Perf: jscpd hashes tokens directly (FNV-1a 64) instead of interning through a shared map, so window hashing runs on the parallel pool — 2.3x faster detection at half the memory.
- Docs: both changes proven behavior-identical — a 216k-word differential test for the spell engine and byte-identical clone statistics on two corpora.
- Fix: the earlier "27ms process baseline" was a shell-timer artifact (node startup inside the measured window); the real baseline is ~3ms and needed no work.

### 2026.07.18.1821

- Perf: parallelize file scanning inside the heavy checks — the full 18-check dogfood suite drops from ~111ms to ~68-76ms.
- Feat: new `internal/par` worker pool — generic and deterministic, results return in input order, so parallelism can never change a check's output; pinned by ordering tests.
- Refactor: the pool now backs `walkfs.ScanFiles` (every markdown scanner and `no-eslint-disable`), the cspell per-file loop, jscpd's read-and-lex phase (clone detection stays serial — it builds shared hash tables), and `go-complexity` parsing.
- Chore: `go-complexity` flagged the new pool's own `Map` at 6 before it could land — the clamp logic became a helper.

### 2026.07.18.1819

- Feat: add the `go-complexity` check — the first net-new check of the Go era and the Go-native counterpart of the house eslint rule (`complexity: max 5`), pure standard library (`go/ast` + `go/parser`).
- Feat: scoring matches eslint — every function starts at 1 and gains a point per `if`/`for`/`range`/non-default `switch` or `select` clause/`&&`/`||`; function literals score separately; `_test.go` files are exempt; the ceiling is configurable via `goComplexity.max`.
- Chore: opt-in (Go-specific, like `swiftlint`) and enabled on this repo — all 50 Go files scanned in about 10ms.
- Refactor: burn every one of the 78 flagged functions down under the ceiling — behavior-identical helper extractions across the runner, the vitestconf and clonedetect lexers (complexity 28 and 25 at the worst), the spell engine, the mermaid parser, and 24 check binaries; 40 packages stay green with no test expectations touched.

### 2026.07.18.1749

- Docs: bring every document in line with the Go-only, npm-free reality — all 27 check READMEs correct their era, with rule documentation, error formats, and examples preserved byte-identical throughout.
- Docs: `.fitnessrc.js`/`.ts` snippets become `.fitnessrc.json`, `npx fitness` becomes `fitness`, "bundled as a dependency" claims become the peer-tool exec or native-engine truth, and TypeScript internals become their Go equivalents.
- Docs: config-fallback descriptions now state the three-step resolution (repo-local, installed `@mayjournal/fitness-shared`, embedded copy materialized on demand); dead `.cursor/rules` pointers are dropped.
- Docs: the architecture docs drop the last stale claims, `competition.md` contrasts against the static check-binary catalog, and `architecture-index.md` leads with the two completed milestone plans; the ADR is deliberately untouched — it is a dated decision record.

### 2026.07.18.1742

- Chore: consolidate all documentation under `docs/` — `architecture/`, `plans/` (with its archive), `architecture-index.md`, `wardley.md`, and `competition.md` move together, so every doc-to-doc relative link survives unchanged.
- Fix: front matter `relatedConfigurations` paths deepen one level in 17 files and README links cross the new boundary.
- Chore: the `mermaid-level-bleed` check keeps matching level files at `docs/architecture/` thanks to its unanchored path pattern; a stale npm-era phrase in `competition.md` refreshed.

### 2026.07.18.1721

- Fix: `go build -o bin` fails when the gitignored `bin/` directory does not exist yet, breaking fresh clones, the pre-commit hook, and CI alike.
- Build: every build command — hooks, CI, README, architecture docs — now runs `mkdir -p bin` first.
- Test: caught by the clean-clone simulation; clone, build, and the full suite now verify green from an empty checkout.

### 2026.07.18.1720

- Feat: the repo is npm-free — plan 02 complete and archived; cloning and building requires exactly one tool: Go.
- Feat: the shared tool configs live inside the binaries — `go/internal/sharedconf` embeds the config directory and materializes it to a content-keyed cache dir on demand; resolution is local config, then an installed `@mayjournal/fitness-shared` for compatibility, then the embedded copy — 153 new tests across the six touched packages.
- Feat: the pre-commit stamping tool is Go (`fitness-stamp-changelog`: restamps the first heading when CHANGELOG.md is staged, re-stages, no version bumps — the changelog timestamp is the version now), and the githooks call `go build` plus the binaries directly.
- Chore: deleted package.json, package-lock.json, node_modules, .npmrc, .nvmrc, every node script under `scripts/`, the npm publish workflows, the node CI setup action, and the `@mayjournal/fitness-shared` package directory (published versions keep winning over the embedded fallback when installed); CI is two Go jobs.
- Chore: the spell dictionaries are frozen committed data (a Go regeneration tool fetching sources over HTTPS is deferred); the dogfood list drops eslint, prettier, node-version, and dependency-currency (nothing left for them to judge here; all four stay in the catalog), leaving 17 checks.

### 2026.07.18.1556

- Feat: retire the TypeScript implementation — the Go suite is now the only fitness runner.
- Chore: deleted `packages/runner`, `packages/checks` (all 26 TypeScript check packages), `packages/checks-bundle`, the legacy `.fitnessrc.js`, and the go-parity harness (26/27 byte-parity was proven before deletion; see the archived plan).
- Docs: each check's rule documentation moved to its Go home (`go/cmd/fitness-check-<name>/README.md`, front-matter paths rebased), the `read-repo-first` banner points there, and README plus the architecture C4 docs are rewritten for the Go-only world.
- Refactor: `packages/shared` survives as a configs-only npm package — eslint, prettier, vitest, and cspell configs consumed as data; build machinery, bin scripts, and runtime sources removed; the unmet optional vitest peer dropped so dependency-currency stays quiet.
- Chore: root scripts slim to `npm run fitness` (Go suite) and `npm test` (`go test ./...`); CI drops the fitness-ts and go-parity jobs; devDependencies shrink to the lint/format stacks plus `cspell` + `cspell-trie-lib` (kept solely to regenerate the embedded dictionaries); the vitest coverage checks leave this repo's list (both stay in the catalog).

### 2026.07.18.1524

- Chore: merge the go-rewrite branch to main — a fast-forward (the branch was strictly ahead), so no merge commit and no hook exception needed.
- Docs: archive the completed milestone plan to `plans/archive/01-go-rewrite.md` with `status: completed` front matter per the plan-doc convention.
- Docs: the README Go-runner section now links to the archived plan.

### 2026.07.18.1521

- Feat: land plan 01 section 5 — lock it in; plan 01 is fully checked off.
- Feat: the go-parity harness (`npm run parity:go`, `scripts/go-parity/`) diffs every check name discovered from the built Go binaries against its TypeScript twin on this repo — ok, errors with path normalization, filesChecked with the documented jscpd exemption — and 27/27 agree.
- Ci: CI gains `go` (gofmt/vet/build/test), `go-parity`, and `fitness-ts` jobs; the `fitness` job now runs the Go suite.
- Feat: dogfood cutover — `.fitnessrc.json` carries the full 23-check list and `npm run fitness` builds and runs the Go runner (~1.9s vs ~6.5s plus a build for `npm run fitness:ts`, which stays supported and CI-gated during the transition).
- Chore: distribution decided in the README (GitHub Releases plus `go install` first; an npm shim only if consumers want npx continuity); `jscpd` joins the repeated-string-literals allow baseline in both configs — CLI binary name, check name, and count-semantics exemption, one occurrence per island.

### 2026.07.18.1511

- Feat: land plan 01 section 4 — the four tool-exec checks, completing the 27-name Go check catalog; every one execs the real tool as a peer, resolved from `node_modules/.bin` walking up, then PATH — never npx — with one-line install hints when missing.
- Feat: `prettier` ports the staged filtering, glob mode, passthrough (`--write .` verified), and `[warn]` parsing; `eslint` execs the CLI with the shared flat config and matches the TypeScript check byte-for-byte on this repo (242 files) and on rule-violation fixtures.
- Feat: `vitest-coverage-full` reuses `internal/vitestconf` for the threshold gate (22 new cases) then execs `vitest run --coverage` with the TS config fallback and a 120-second describe budget; `swiftlint` execs the system binary with real JSON violation parsing, byte-identical against swiftlint 0.65.0.
- Test: all four verified side-by-side with the real tools plus scripted-fake mocks in `go test` (122 new cases), including the missing-tool paths.
- Chore: full-catalog sweep — 26 of 27 check names byte-match the TypeScript twins on this repo; the one difference is `jscpd` scanned-file-count semantics, documented in section 3, with verdict parity.

### 2026.07.18.1312

- Feat: `cspell` is now a Go binary over `internal/spell` — camelCase-aware word extraction, cspell-compatible inline directives, default URL/email/hash/escape masks, and 14 committed plain-text wordlists (~217k entries, 2.1MB) generated from the installed `@cspell` packages by `go/internal/spell/dict/generate.mjs` (provenance in every header).
- Test: spell parity — byte-identical to the real cspell CLI on a 31-issue adversarial probe corpus and a clean 258-file tracked-source sweep; this repo passes with the exact TypeScript file count (52), about 7x faster.
- Feat: `jscpd` is now a Go binary over `internal/clonedetect` — a generic comment-stripping lexer, rolling-hash windows with jscpd's min-lines/min-tokens/threshold semantics, the `jscpd:ignore-start`/`end` escape hatch, batched `git check-ignore` filtering, and the verbatim over-threshold error line.
- Test: clone-detection parity — string literals keep their content in the token stream (full collapse falsely merged the 21 check mains' boilerplate); verdicts agree with the TypeScript check on this repo and on seeded-clone, ignore-marker, and below-threshold fixtures; 165 new Go tests.
- Fix: the `cspell` check hung the pre-commit hook when the 2.1MB wordlists were staged — cspell applies ignorePaths only to discovered files, never explicit arguments — so the staged filter now drops any staged path a non-glob ignorePaths entry covers; `go/internal/spell/dict` joins the shared ignorePaths.

### 2026.07.18.1208

- Feat: land plan 01 section 2 — the parsers-and-network checks are now Go binaries; all seven side-by-side comparisons against the TypeScript twins agree, passing with identical file counts.
- Feat: `internal/mermaid` ports `mermaid.ts` exactly — fence scanning, the five callout patterns with JS-lookahead emulation over RE2, GFM callout tables, legend-invisible pairing — pinned by 47 tests (12 emulation edge cases executed against the real JavaScript first) and a byte-for-byte differential dump over all 52 repo markdown files.
- Feat: the five mermaid checks are thin binaries over that parser; `vitest-coverage-exclude` scans vitest config sources text-level via string-aware comment stripping in `internal/vitestconf` (shared home for the future coverage-full port), with the TS fallback-root semantics.
- Feat: `dependency-currency` replaces the `npm outdated` shell-out with a native net/http registry client — abbreviated-metadata endpoint, `.npmrc` registry honored, bounded concurrency, offline and garbage responses degrade to pass exactly like the TypeScript check — ~2.6x faster than the npm oracle on this repo.

### 2026.07.18.1140

- Feat: land plan 01 section 1 — all thirteen pure-logic checks are now Go binaries, twelve ported in one parallel pass, the markdown-filename pair as two thin binaries over one shared `internal/mdfilename` package.
- Test: each port carries table-driven tests from the meaningful TypeScript cases (~200 Go cases total) and is proven side-by-side against its twin on this repo — passing checks pass identically, failing checks fail with byte-identical errors.
- Feat: `changelog` upgrades to real JSON parsing for the invalid-JSON error paths; `semantic-commit` and `commit-attribution` declare the `--message` context-inline handshake; `repeated-string-literals` reads its allow list from `.fitnessrc.json`.
- Chore: add `.fitnessrc.json` carrying the `repeated-string-literals` allow baseline for the Go runner during the config transition (the TypeScript suite keeps reading `.fitnessrc.js`; both lists stay in sync until the dogfood cutover).
- Fix: `jscpd` now also ignores Go test files (`**/*_test.go`) — the test/spec exclusion rationale predates the Go tree; the two real production clones the ports introduced were extracted instead (a shared `walkfs.ScanFiles` scan loop and the `render.ColorsEnabled` color gate).

### 2026.07.18.1113

- Fix: the `prettier` check errored on staged Go files — Prettier has no parser for `.go` or `go.mod`, so the first commit carrying the new `go/` tree failed pre-commit; staged paths under `go/` are now dropped from the staged-mode invocation (same treatment as `scripts/` and `githooks/`).
- Feat: land plan 01 section 0 — the Go scaffold: a new stdlib-only `go/` module with the `fitness` runner and the first check binary, `fitness-check-node-version`, running end to end on this repo.
- Feat: the runner resolves check binaries (sibling dir then PATH, local paths from config), execs them with the `--root` + `FITNESS_*` env protocol (JSON result on stdout, display on stderr), budgets each `--describe` handshake at two seconds, and kills timed-out checks by process group from a bounded pool — same table and exit codes as the TypeScript runner.
- Feat: config is `.fitnessrc.json`; a lone legacy `.fitnessrc.js`/`.ts` gets a migration hint on full-suite runs only, so single-check runs work during the transition; shared internals (skip-dir walker, git helpers, markdown parsing, table renderer) all carry `go test` coverage.
- Test: the Go `node-version` check agrees with the TypeScript check on this repo — identical table row and byte-identical failure message; its checkbox and all of section 0 are flipped in `plans/01-go-rewrite.md`.

### 2026.07.18.1057

- Docs: add `plans/01-go-rewrite.md` — the milestone plan for rebuilding the runner and every check in Go as zero-dependency static binaries: one binary per check plus a `fitness` runner, stdlib-only, exec protocol with JSON results.
- Docs: checks port one at a time with side-by-side parity against the TypeScript checks; the dep-heavy four (prettier, eslint, vitest-coverage-full, swiftlint) come last with each approach decided on arrival.
- Docs: the plan follows the may-journals template — numbered title, Goal, numbered checkbox sections, verification last.
- Chore: bump `@typescript-eslint/eslint-plugin` + `@typescript-eslint/parser` 8.63.0 → 8.64.0, `eslint-plugin-jsdoc` 63.0.13 → 63.1.0, and `knip` 6.26.0 → 6.27.0 to satisfy `dependency-currency`; full build/fitness/lint suite verified on the updated tree.

### 2026.07.10.2041

- Feat: add the `repeated-string-literals` check — the same literal appearing 3+ times across source files fails, most-repeated first, with extract-a-constant guidance; comments, regexes, template literals, import specifiers, idiomatic tokens, directives (`'use strict'`), and test/spec/bench files are never flagged. Closes #42.
- Feat: derive the bundler's check list from the `packages/checks/*` directories instead of a hand-maintained array — a directory bundles as a same-named check unless its package.json maps entry modules via `fitnessChecks` (flavor packs). Closes #41.
- Feat: add the `commit-attribution` check — commit messages must disclose AI usage via `AI-Tools:` and `AI-Models:` trailers after the subject; merge and revert commits exempt; opt-in, with the README expanded to the passing/failing/advanced standard. Closes #7.
- Fix: the `eslint` check no longer builds a TypeScript type-checker program — no enabled rule reads types, so it cost ~5s for zero findings and tipped CI past the timeout; syntactic parsing is byte-identical and ~4x faster, recorded as ADR 0001 (`fix-the-work-not-the-limit`); `build-output-untracked` stops flagging its own test fixtures.
- Chore: dogfood both new checks, fix every repeated literal found (allow baseline 55 → 3 via shared constants and enum-valued check names), add the `repeatedStringLiterals.allow` option (inline marker and enum heuristic deferred to #44), bump the toolchain (TypeScript 6.0.3 → 7.0.2, eslint 10.7.0), and merge `main` into the branch.

### 2026.07.07.0850

- Feat: add the check packages `markdown-filename-convention` (kebab-case + camelCase flavors over one shared parameterized function, #11), `no-eslint-disable` (#10), `gitignore-why` (#6), `build-output-untracked` (#8), the five-check mermaid diagram + callout-table set, `dependency-currency` (#30), and `jscpd` + `swiftlint` modeled on may-journals' setup.
- Feat: `.fitnessrc` `checks` accepts local module paths mixed with npm check names, in order — missing or invalid path entries fail the run, `disabledChecks` never removes them, and path loading is shared between `--check=./foo.js` and config (#23).
- Fix: `dependency-currency` timed out in CI (~5.5s on a cold registry cache) — checks gain an optional `timeoutMs` the runner honors (30s here); `bundle-check-dist.mjs` always rebuilds and clears its destination so a stale bundled `dist` cannot ship; `bin/fitness.js` drops its cwd override so `npx fitness` loads the consumer's `.fitnessrc`.
- Chore: bring every dependency to latest and enable the full suite here — TypeScript 6, ESLint 10 (shared config to ESM; `typescript-sort-keys` replaced by `perfectionist`), cspell 10, Vitest 4.1, knip 6; `jscpd` re-enabled (test/spec ignored); check dependencies propagated to published packages, with `boxen` dropped entirely (#17).
- Build: publish only `@mayjournal/fitness-shared`, `fitness-checks`, and `fitness` (checks bundled; individual workspaces private) with npm OIDC trusted publishing, provenance, and sequential workspace publish; Husky replaced by native Git hooks in `githooks/`; C4 architecture levels 1-5, a Wardley map, and a competition matrix land under `architecture/`.

### 2026.05.20.1716

- Perf: land the publish-audit plan (#13) — an `audit:publish` script, publish-audit CI with PR comments, `bench:load-check`, per-workspace knip, and types-first `exports` on the publishable packages.
- Fix: publish-audit CI — track `packages/shared/types/vitest.config.d.ts` for publint, drop broken attw from the PR workflow, strip ANSI from publint output, fix the `findInstallRoot` test for hoisted vs workspace `node_modules`, and write the audit JSON via `node scripts/audit-publish/index.mjs` so PR comments parse.
- Feat: add JS provision scripts (per-folder READMEs) and publish-time checks that seed new `@mayjournal` workspaces and configure trusted publishing — then slimmed: the Provision workflow removed, Publish runs OIDC `npm publish -ws` only, optional local helpers kept, and `test:scripts` adds colocated node:test coverage.
- Chore: dependency and config hygiene — remove the redundant runner jiti, duplicate prettier plugins, and misplaced production deps on checks; a single cspell config at `@mayjournal/fitness-shared/cspell` (duplicate removed); `repository` added to publishable package.json files.
- Build: `@mayjournal/fitness-shared` and `@mayjournal/fitness-checks` build via `fitness-shared build` (inline `node --eval` tsconfig generation removed); provision and related scripts formatted with Prettier for CI.

### 2026.05.20.1516

- Fix: `fitness-shared` build, lint, and test resolve the monorepo root internally.
- Refactor: remove `cd` from the package scripts now that the bins resolve the root themselves.
- Fix: generate an ephemeral runner `tsconfig.json` at lint time so ESLint `projectService` maps runner sources on clean checkouts.

### 2026.05.20.1509

- Refactor: centralize ESLint rules in `eslint.base.cjs` — the CLI and the fitness eslint check both import `createEslintConfig` with their own TypeScript parser options, so `projectService` and `project` no longer conflict.
- Feat: add `fitness-shared lint` for monorepo ESLint; workspace lint scripts route through it; pre-commit runs lint alongside fitness.
- Chore: check packages get a minimal lint-only `tsconfig.json` (extends the shared check config) for `projectService` discovery; monorepo check enumeration leaves the shared config.

### 2026.05.20.1453

- Feat: split the repo into npm workspaces — runner and checks move from root `src/` into `packages/runner` and twelve `packages/checks/*` workspaces with dynamic check loading and `@mayjournal/fitness-checks-bundle` defaults; per-check tool deps live in each check's package; the plan completes through release (PR #12); `Architecture.md` documents the layout.
- Feat: checks fall back to `@mayjournal/fitness` configs when consumers lack local cspell, prettier, vitest, or tsconfig — `resolveFitnessConfigPath`, `resolveLintTsconfig` (temp tsconfig for ESLint in the parent cwd), and a `tsconfig.lint.cjs` export; consumer setup documented in README.
- Feat: restore `disabledChecks` on `.fitnessrc` — the optional list removes names from an explicit `checks` list or from bundle `defaultChecks` after `resolveCheckNames`, typed in `fitness-shared`, filtered in `load-check.ts`, covered by tests.
- Build: centralize TypeScript `compilerOptions` in `tsconfig.compiler.cjs`, generate check and bundle tsconfigs from CJS sources, commit them for CI, and build `fitness-shared` first so runner `tsc` and type-aware ESLint resolve monorepo sources; publishing switches to npm trusted publishing (OIDC) with `NODE_AUTH_TOKEN`.
- Chore: CI runs workspace tests (`-ws --if-present`); Vitest is scoped per check package with runner-dist aliases; `changelog-updated` expects the heading to match the root version suffix after long CI runs; staged cspell honors `ignorePaths`; `ensure-changelog-timestamp.cjs` bumps versions under `packages/`.

### 2026.04.04.1748

- Feat: export `./vitest.config` as `vitest.config.mjs`.
- Feat: add `vitest.config.d.ts` for TypeScript consumers.
- Fix: resolve Prettier plugin paths with `createRequire` so consumers loading `@mayjournal/fitness/prettier.config` resolve plugins from this package.

### 2026.04.04.1712

- Feat: the ESLint check runs via the Node API with this package's `eslint.config.cjs` and the parent project cwd, so consumers need not install ESLint — adds `getFitnessRunnerRoot` (shared with the runner and vitest-coverage-full), `runInProcess`, and `RunContext._eslintRunForTesting`.
- Fix: show `file:line:col` errors by extracting the JSON array when stderr is mixed in; on parse failure, append truncated ESLint output to the fallback; `tryParseJsonArray` keeps complexity under the limit.
- Build: scope the package to `@mayjournal/fitness` — LICENSE (MIT), a GitHub Actions publish workflow on CI success, a `publish:ci` script, and `.npmrc` for `NPM_TOKEN`; `plan-deps-vs-devdeps-check.md` added.

### 2026.03.07.1922

- Chore: Prettier uses `prettier-plugin-packagejson` for conventional `package.json` field order; `package-lock.json` joins `.prettierignore`.
- Refactor: single source of truth for check registration — optional `Check.folder` plus `RunContext.checkFolderByName` (worker-serializable) built from the registry; `CHECK_TO_FOLDER` leaves read-repo-first; `registry.test.ts` asserts the `CheckName` enum and registry stay in sync; add-a-check steps documented.
- Refactor: shared `quoteForShell` (`src/utils/shellQuote.ts`) used by eslint, prettier, and cspell; a shared Vitest config loader (`src/checks/vitest-config`) used by both coverage checks.
- Refactor: shared exec helpers — `execSyncResult()` with `EXEC_OPTS`, and `buildExecCheckResult()` — adopted by eslint, prettier, cspell, vitest-coverage-full, and changelog-updated.
- Fix: `markdown-no-bold-italic` ignores emphasis inside link blocks `[text](url)` so underscores in URLs or link text are not falsely flagged.

### 2026.03.07.1431

- Fix: the runner dedupes `config.checks` by name — `checksFromConfigList` keeps the first occurrence, so a check listed multiple times in `.fitnessrc` runs once.
- Test: cover duplicate names in `config.checks` and add a re-entry guard test.
- Refactor: extract `runImpl` to satisfy the eslint complexity ceiling.

### 2026.03.07.1406

- Feat: ESLint gains the `max-lines` rule (200, skipBlankLines/skipComments).
- Refactor: split `run.ts` into `run-resolve.ts` (getChecks, config/spec resolution), `run-execute.ts` (runOneCheck, worker/in-process), and `run-output.ts` (buildTable, buildTotalLine).
- Refactor: `run.ts` keeps orchestration only.

### 2026.03.07.1007

- Feat: the runner executes registry checks in a worker thread so the 5s timeout is enforced via `worker.terminate()` when checks block (e.g. execSync); read-repo-first, vitest-coverage-full, and path-based checks stay in-process; timed-out checks fail with "Check timed out after 5s" and the run continues.
- Feat: `node_modules` is always ignored — `getSkipDirs` merges the runner skip dirs (`node_modules`, dist, coverage, .git, .husky) with config, and staged files from git are filtered the same way.
- Feat: progress messages on stderr (Resolving checks…, Running checks:, and → name before each check) show where a run is or where it hangs.
- Feat: add `vitest-coverage-full` (runs `vitest run --coverage`; requires 100% thresholds); cspell gains a `runCspell` CLI runner, the enUS dictionary, and a runViaExec/runViaLib split; `vitest-coverage-exclude` allows barrel `index.ts` excludes and recognizes `vitest.config.cjs`.
- Fix: `isMainModule` works when run via npx — argv[1] and `import.meta.url` resolve to real paths so the symlinked `.bin/fitness` is detected as main; a symlink test, JSDoc, and complexity cleanups ride along.

### 2026.02.22.1620

- Feat: add `checkResult(ok, errors?, filesChecked?)` and a `runContext` helper (getStagedFiles, getExecSync); every check migrates to them; `RunContext` gains `_now` for tests.
- Docs: add `plan-checks-abstractions.md` (the repeating patterns in `checks/*` and abstraction options); the README flowchart node renamed to PassthroughArgs; the cspell words list trimmed.
- Build: flatten dependencies into `dependencies` only (no dev/optional split).
- Chore: Prettier overrides `package.json` to the json parser so sort-json runs recursively over exports paths and condition keys; `prettier-plugin-packagejson` removed; comments added in `prettier.config.cjs`.
- Refactor: an `enUS` enum carries all user-facing runner copy (run logic in `run.ts`, `index.ts` barrel only), with an `interpolate()` util for `{{key}}` templates, exported from runner and package; eslint adds `typescript-sort-keys` (string-enum + interface) with `@typescript-eslint` aligned to ^8.55.

### 2026.02.22.1511

- Refactor: Prettier collapses to a single config (`prettier.config.cjs`); `.prettierrc.json` removed.
- Feat: add `.prettierignore` and `prettier.config.d.ts` with a package.json types export.
- Fix: the Prettier check recognizes `.ts`/`.mts`/`.cts` config names.
- Chore: gitignore the generated Prettier files and drop the unsupported `ignore` option from the config.

### 2026.02.22.1454

- Fix: `semantic-commit` fails when there is no message to validate (empty or git unavailable).
- Feat: the commit-msg hook passes message content via `--message="$(cat "$1")"`.
- Chore: export `MSG_EMPTY`.

### 2026.02.22.1431

- Refactor: replace `findMd` with `findFilesByExtension(root, extension)`.
- Feat: `getSkipDirs` uses `skipTheseDirectories` from `.fitnessrc` when present, else cspell.json `ignorePaths` (dir names only); no default list.
- Feat: add `FitnessConfig.skipTheseDirectories`.
- Chore: add `.git` and `.husky` to cspell.json.

### 2026.02.22.1408

- Feat: add the `CheckName` enum; all checks use it for `name` (no static strings).
- Refactor: the runner casts config check names to `CheckName` for registry lookup.
- Chore: export `CheckName` from types.

### 2026.02.22.1359

- Fix: make the README Mermaid flowchart edge labels readable — the theme gains `textColor` and `labelColor` so the yes/no arrow labels stop blending into the background.
- Docs: reroute the context edges — `buildContext(staged, inlineFragment, checks, passthrough)` now feeds `runChecks`, and the single-check `contextInline` fragment hangs off the resolved checks.
- Docs: annotate `getStagedContext()` with its mechanism (`git diff --cached` → stagedFiles).

### 2026.02.22.1332

- Feat: the runner uses check-registered `contextInline` — no check-name logic remains in the runner.
- Feat: the `Check` type gains optional `contextInline` (argName, contextKey); `ContextInline` is exported.
- Feat: `semantic-commit` registers `--message` → `proposedCommitMessage`; the commit-msg hook uses `--message="$(cat "$1")"`.
- Docs: update README and the flow diagram.

### 2026.02.22.1322

- Docs: the runner-no-check-names plan lands on check-registered context — `contextInline` (and a future `contextPath`) on the `Check` type.
- Docs: `.fitnessrc` needs no change under the chosen approach.
- Docs: the plan doc slims from exploratory options down to the decision (139 lines removed).

### 2026.02.22.1259

- Refactor: rename `specFromPositional` to `checkNameIsFirstArg` and `getCommitMsgPath` to `getContextFilePath` for clarity.
- Docs: add `plans/plan-runner-no-check-names.md` with front matter.
- Style: no bold/italic in the new plan, for the markdown checks.

### 2026.02.22.1247

- Refactor: abstract check dependencies out of the runner — `getColumns` moves to `src/utils/terminal` so the runner has no check-specific imports.
- Refactor: `read-repo-first` no longer exports `getColumns`.
- Test: the `getColumns` tests move to `utils/terminal.test.ts`.

### 2026.02.22.1241

- Docs: align the README Mermaid code-flow diagram with the real runner flow.
- Docs: nodes now name the actual functions — `resolveCheckSpec`, `resolveChecksBySpec`, `buildContext`.
- Docs: the spec-defined branch appears in the flow.

### 2026.02.16.1646

- Feat: the `changelog` check requires the `package.json` version suffix to match the first `###` heading (`yyyy.mm.dd.HHMM`).
- Feat: the `package-lock.json` version must match the same heading.
- Test: extend `changelog.test.ts` to cover both version gates.
- Docs: update the check README.

### 2026.02.16.1634

- Feat: `changelog-updated` requires the new section heading to use the current date and time (`yyyy.mm.dd.HHMM`) so GenAI cannot guess the time.
- Test: add current-time heading cases to `changelog-updated.test.ts`.
- Docs: update the check README for the current-time rule.

### 2026.02.16.1900

- Refactor: `changelog-updated` exports its human-facing message consts (`MSG_*`).
- Refactor: the implementation reuses the exported consts — one source for the copy.
- Test: tests assert against the exported consts instead of duplicating strings.

### 2026.02.16.1800

- Feat: the runner adds `passthroughArgs` to `RunContext` when running a single check.
- Feat: args after the check name forward to checks (e.g. `npx fitness prettier --write`).
- Feat: the Prettier check runs Prettier with forwarded args instead of `--check` when present.

### 2026.02.16.1700

- Build: move eslint, prettier, vitest, and the related config plugins from devDependencies to dependencies so consumers can use the exported configs.
- Build: keep `@types/node`, `tsconfig.js`, tsx, and typescript as devDependencies.
- Feat: the Prettier check detects config via the package.json `"prettier"` field so consumers using `"prettier": "@mayjournal/fitness/prettier.config"` are checked.

### 2026.02.16.1600

- Feat: ESLint adds `sort-keys` (natural ascending) for all object keys in `.ts`, `.cjs`, `.js`, `.mjs`; the ESLint check extends to `.cjs`/`.js`/`.mjs`; object literals reordered across the codebase.
- Feat: add Prettier with `eslint-config-prettier` — `.prettierrc.json`, `.prettierignore`, format scripts, a CI format job, and an exported `prettier.config`; `prettier-plugin-sort-json` with jsonRecursiveSort orders JSON keys and nested objects; the Prettier check runs `prettier --check` on staged files or the full repo and skips when no config.
- Feat: the runner renders table feedback — a colspan row per failed check, errors contextual to the row, dynamic width via `getColumns` from read-repo-first.
- Docs: sync the main and checks READMEs with the registry; add the rules-front-matter README and `fitnessFunctions` to the eslint-check README; update the README flow diagram.
- Fix: `rules-front-matter` rejects empty `fitnessFunctions` and `relatedConfigurations` arrays — at least one entry per array.

### 2026.02.16.1500

- Feat: the runner formats check results as a table (cli-table3) — Check, Status, Files, Time columns, bold white headers, chalk green/red for results, errors, and the total line.
- Feat: `read-repo-first` prompts Y/N to confirm familiarity with the checks, lists the enabled ones in a table with a plain-path Src column for IDE link detection, formats with chalk/boxen/wrap-ansi, and runs first in the registry.
- Fix: `read-repo-first` drops its TTY requirement — feedback displays for Agent/User contexts and always passes; the `FITNESS_READ_REPO_CI_ONLY_DO_NOT_USE_OTHERWISE` bypass leaves CI.
- Chore: consolidate the cursor rules into `fitness-checks.mdc`, removing the per-check rule files.
- Chore: add `eslint.config.d.ts` for ESM package compatibility; package and gitignore updates for the consolidated rules.

### 2026.02.16.1400

- Feat: add `package.json` exports so downstream projects can reuse the shared configs.
- Feat: the exported entries — `./eslint.config`, `./vitest.config`, `./tsconfig.cjs`, and `./cspell` (raw `cspell.json` data).
- Build: ship the config files in the npm package — the `files` list adds them beside `dist`.

### 2026.02.16.1300

- Build: a single root `tsconfig.cjs` drives build and lint — the shared base config removed, compiler options inlined, `build/` gone, and `tsconfig.js` converts `.cjs` to JSON; generated root config outputs are gitignored.
- Refactor: replace `eslint.config.ts` with `eslint.config.cjs` for ESM package compatibility.
- Chore: ESLint drops the stylistic plugin and rules, keeping `jsdoc/require-jsdoc` and `complexity` max 5 in a single block with project tsconfig.json.
- Fix: `changelog-updated` types the `ExecSyncFn` maxBuffer and adds an execSync fallback coverage test.
- Docs: the `vitest-coverage-exclude` README uses relative paths and is properly associated with the name of the check.

### 2026.02.16.1025

- Feat: the ESLint check runs eslint (staged paths or `.`), parses its JSON output, and reports errors.
- Fix: only `.ts`/`.tsx` staged paths are passed, avoiding no-config failures on `.md`; tests use `.ts` fixtures (bar, pathWithQuote, quoted).
- Refactor: hoist the feedback and CLI consts.
- Test: 100% coverage on the check.

### 2026.02.16.1005

- Feat: the runner adds feedback dressing — "Please fix these items."
- Refactor: hoist the messages to shared consts.
- Test: runner tests use a static import.

### 2026.02.16.0958

- Docs: add the `read-repo-first` check design doc (`plans/read-repo-first-check.md`).
- Docs: the problem is framed as machine-checkable proxies for a behavioral rule, with four design directions — structure-only validation, staged correlation, advisory no-op, and a configurable hybrid.
- Docs: draft recommendation — start with structure-only validation (numbered folders at the repo root) as the minimal viable check.

### 2026.02.15.1700

- Docs: clarify the README tagline (fitness runner, checks, workflows).
- Perf: run cspell in-process via cspell-lib (readConfigFile, spellCheckFile) for speed; the CLI path remains when tests mock exec.
- Feat: the summary prints total success and failure counts, rounded time, and total files scanned (the sum of `filesChecked` from checks).
- Fix: `markdown-no-bold-italic` no longer flags unordered-list asterisk markers as italic.

### 2026.02.15.1600

- Docs: remove bold/italic from the check READMEs to satisfy `markdown-no-bold-italic`; README and CHANGELOG front matter fixed (---, flow-style).
- Feat: `markdown-front-matter` requires `fitnessFunctions` or `relatedConfigurations` in every `.md`, paths resolved relative to the file — or any registered check name; `findMd` skips `node_modules`, dist, coverage, .git, .husky; `getFrontMatterPaths` exported for tests (100% coverage).
- Feat: `vitest-coverage-exclude` only allows `**/*.d.ts` and `**/*.types.ts` in coverage exclude (Vitest excludes tests by default); type-only files renamed to `*.types.ts`; load.ts and coverage-exclude branches fixed for full coverage.
- Chore: remove `.fitnessrc.ts`, with a runner test covering custom checks from a config to restore 100% coverage.
- Ci: a single fitness job runs `npm run fitness` — the discover job and matrix removed, checks listed as single-line GITHUB_OUTPUT for valid JSON; pre-commit sources nvm in the husky hook so `nvm use` runs when PATH has no nvm.

### 2026.02.15.1500

- Refactor: the node-version check compares `.nvmrc` to the current Node only — the earlier nvm-subshell validation (with a process.version fallback) is removed.
- Ci: the CI script runs `nvm use` when available.
- Fix: default the strategy matrix `fromJson(needs.discover.outputs.checks)` to `'[]'` when the checks output is empty.

### 2026.02.15.1400

- Test: the runner reaches 100% coverage — path-load tests for the named-export, no-Check, and import-throws cases.
- Feat: two positionals (check then message path) supported for semantic-commit.
- Fix: `getPositionalSpec` and `getCommitMsgContext` handle single vs two positionals.

### 2026.02.15.1300

- Feat: the rules front-matter check validates `fitnessFunctions` and `relatedConfigurations` paths in all markdown.
- Refactor: a shared `findMd` helper backs the markdown checks.
- Feat: the runner accepts a check by name or path (`--check=./path/to/check.js` or positional).
- Feat: a `Check` loads from a module's default or named export.

### 2026.02.15.0100

- Refactor: the runner takes a single CLI flag — `--check=` only.
- Feat: the commit-msg path rides as a positional.
- Feat: staged context is always built.
- Test: full runner test coverage.

### 2026.02.15.1200

- Ci: GitHub Actions runs dynamic fitness jobs from the registry with a composite setup action.
- Feat: add the cspell check — optional, it only runs when `cspell.json` is present; cspell lives in runner dependencies only, with the spell script removed from ci and package.json.
- Docs: add a Mermaid code-flow diagram to the README (modern colors, moved to the bottom); cspell README updates.

### 2026.02.15.1100

- Feat: a commit-msg hook runs `semantic-commit`; the check merges into a single file; cursor rules point to the check READMEs.
- Feat: the `changelog` check requires `### yyyy.mm.dd.HHMM` — every `###` heading must match, and the section format matches the package version.
- Feat: `changelog-updated` suggests up to 10 random words from the staged diff when overlap is too low.
- Chore: remove the duplicate check-node-version script and check-node; ci runs `npm run fitness` only.

### 2026.02.15.1000

- Feat: add the fitness config (`.fitnessrc.ts`) and config loader; turn on all checks (changelog, semantic-commit).
- Feat: add the `changelog-updated` check — fuzzy-matches the staged diff to the changelog, checking only changed lines, not the whole file.
- Refactor: colocate tests with source; merge the runner tests.

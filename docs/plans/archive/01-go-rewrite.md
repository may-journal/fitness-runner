---
relatedConfigurations: ['../../../.fitnessrc.json']
status: completed
completedAt: 2026-07-18
---

# 01 — Go rewrite: the whole suite as static binaries

> Node, a build step, and a dependency tree just to run a check. Rebuild it in Go — one static binary per check plus a runner, zero dependencies, ported one check at a time.

## Goal

`fitness` runs the full suite on any repo — local dev, git hooks, any CI image — as self-contained static binaries: nothing to install beyond dropping them on PATH, no Node, no build step, instant startup. Every check keeps its soul: same rule, same pass/fail judgment, same error vocabulary as the TypeScript checks in [architecture/03-components.md](../../architecture/03-components.md). Ports land one check at a time, each proven against the TS check on this repo before its box flips; the TypeScript packages stay untouched until the Go suite reaches parity. Dep-heavy checks (prettier, eslint, vitest-coverage-full, swiftlint) come last, each approach decided only when we get there.

## Plan

0. Scaffold: a runner and one working check

   - [x] `go/` module builds `fitness` plus a hello check binary; `fitness node-version` runs it end to end on this repo
     - [x] Stdlib only — no third-party Go imports anywhere in the module
     - [x] One binary per check (`fitness-check-<name>`), runner execs them: `--root <dir>`, context via `FITNESS_*` env (staged files, enabled checks, commit message), JSON result on stdout (`ok`, `errors`, `filesChecked`), human display on stderr
     - [x] `--describe` handshake reports name, timeout budget, and context-inline arg — no registry, no header parsing
     - [x] Runner owns timeouts: process-group kill on expiry so a hung check's whole child tree dies
     - [x] Parallel pool (CPU count), results rendered in dispatch order — same table, totals line, and exit-code contract as today
     - [x] Config is `.fitnessrc.json` (same keys: `checks`, `disabledChecks`, `skipTheseDirectories`, `repeatedStringLiterals.allow`); a lone `.fitnessrc.js`/`.ts` gets a one-line migration hint
     - [x] A local path in `checks` execs any executable speaking the protocol — shell scripts included
   - [x] Shared internals the checks build on: skip-dir file walker, git helpers, markdown front matter/table/fence parsing, results renderer

1. The easy thirteen — pure logic, one at a time

   - [x] node-version — .nvmrc agreement
   - [x] gitignore-why — every ignore pattern carries a why-comment
   - [x] changelog — heading format, version suffix, lockfile agreement
   - [x] changelog-updated — staged-diff word overlap and timestamp gates
     - [x] Preserve the diff-parsing quirks exactly ('+++ ' headers, '+' not '++', CHANGELOG.md root path only)
   - [x] semantic-commit — conventional `type(scope): subject`, scope required
   - [x] commit-attribution — AI-Tools/AI-Models trailers, merge/revert exempt
     - [x] Tri-state message resolution: absent flag falls back to `git log -1`, present-but-empty does not
   - [x] read-repo-first — banner plus enabled-check table through the shared renderer
   - [x] markdown-filename-kebab-case and markdown-filename-camel-case — two thin binaries over one shared `internal/mdfilename` package (sibling-binary resolution wants one binary per name)
   - [x] markdown-front-matter — front matter required; `relatedConfigurations` entries resolve as paths or enabled check names
   - [x] markdown-no-bold-italic — emphasis ban with verbatim snippet quoting
   - [x] no-eslint-disable — directive scan over the source extensions
   - [x] build-output-untracked — dist ignored and untracked, no source imports reaching into dist
   - [x] repeated-string-literals — lexer scan, idiomatic allow set, config allow list

2. Parsers and the network — still native

   - [x] Mermaid parser in `internal/mermaid`: fences, the five callout patterns with position dedupe, GFM callout tables, legend-invisible pairing — pinned by ported tests from `mermaid.test.ts`
   - [x] mermaid-callouts, mermaid-callout-why, mermaid-diagram-prose, mermaid-legend, mermaid-level-bleed — five thin checks over the one parser
   - [x] vitest-coverage-exclude — scan vitest configs for disallowed coverage excludes
   - [x] dependency-currency — native registry client instead of shelling to `npm outdated`
     - [x] net/http against the configured registry (honor `.npmrc`); offline or garbage responses degrade to pass, exactly like today

3. Native souls of two tool checks

   - [x] cspell soul — no unknown words in markdown and staged files; bundled base dictionary plus the project cspell.json `words`/`ignorePaths`
   - [x] jscpd soul — duplicated lines above 1 percent fail; token-normalized clone detection with min-lines/min-tokens semantics and the ignore-marker escape hatch

4. The stubborn four — last, decided on arrival

   - [x] prettier — decided: exec the real binary (node_modules/.bin walk-up then PATH, never npx); staged filtering, passthrough, and error strings ported verbatim
   - [x] eslint — decided: exec the eslint CLI with the shared flat config (a native rule subset would mean a TS parser in Go for a brittle approximation); byte-identical output on this repo
   - [x] vitest-coverage-full — decided: threshold gating via internal/vitestconf, then exec vitest run --coverage with the TS config fallback; 120s describe budget
   - [x] swiftlint — decided: system-binary exec via PATH with real JSON violation parsing; byte-identical against real swiftlint

5. Lock it in

   - [x] Side-by-side harness diffs TS vs Go per check on this repo (`ok`, `errors`, `filesChecked`); a check's box above only flips when it agrees
   - [x] Every check ports the meaningful cases from its TS tests; `go test ./...` green in CI alongside the existing suite
   - [x] Dogfood cutover: `.fitnessrc.json` switches this repo to the Go runner once every enabled check has parity
   - [x] Decide distribution — decided: GitHub Releases + `go install` first, npm platform-binary shim later only if consumers want `npx` continuity; captured in the README Go-runner section

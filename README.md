---
# Top-level project config
relatedConfigurations: ['package.json']
---

# fitness

Zero-dependency Go fitness runner that runs checks for local dev, CI/CD, and GenAI workflows to stay aligned with your intended rules and quality bar.

Every check is its own static binary (`fitness-check-<name>`) orchestrated by a `fitness` runner binary: no runtime dependencies, no build step for consumers, instant startup, parallel execution. The full 21-check suite this repo gates its own commits on runs in under two seconds. The original TypeScript implementation has been retired; its checks were ported one at a time with side-by-side parity before removal (see [plans/archive/01-go-rewrite.md](./plans/archive/01-go-rewrite.md)).

## Install

Build from source today (GitHub Releases with prebuilt binaries are the distribution path once publishing is wired):

```bash
git clone https://github.com/may-journal/fitness-runner && cd fitness-runner
npm run build:go   # or: cd go && go build -o bin ./cmd/...
```

Put `go/bin` on PATH (or copy the binaries somewhere on it). The runner finds check binaries beside itself first, then on PATH.

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

`checks` entries can also be local executable paths (entries containing `/`), mixed in with check names, to run a repo-specific check without publishing anything. A local check is any executable speaking the protocol below — a shell script works.

## Check protocol

The runner invokes each check as `fitness-check-<name> --root <dir> [args…]` with cwd set to the repo root and context in `FITNESS_*` environment variables (staged files, enabled check names, the commit message for the commit checks):

- stdout — one JSON result object: `{"ok": bool, "errors": [".."], "filesChecked": n}`
- stderr — human display output (banners, tool passthrough)
- exit code — 0 when the check ran and passed, 1 ran and failed, other values mean it crashed
- `--describe` — prints check metadata (name, timeout budget, context-inline arg) so the runner needs no registry

The runner executes checks in a bounded parallel pool with per-check timeouts (process-group kill, so a hung check's whole child tree dies) and renders a summary table; the run exits 1 when any check fails.

## Git hooks

This is the point of the runner: checks enforced automatically on `git commit`. This repo's own hooks live under [githooks/](githooks/) — pre-commit runs the full suite, commit-msg validates the message through the semantic-commit check:

```bash
npm run fitness -- --check=semantic-commit --message="$(cat "$1")"
```

Checks that consume the commit message declare a context-inline argument in their `--describe` metadata; the runner extracts `--message` from single-check argv into the environment.

## Usage

```bash
npm run fitness              # full configured suite (builds go/bin first)
go/bin/fitness               # same, without npm
go/bin/fitness prettier      # one check by name
go/bin/fitness --check=eslint
go/bin/fitness prettier --write .   # passthrough args reach the check
```

## Checks

All 27 check names, one binary each under [go/cmd/](go/cmd/), with each check's rule documented in its own README (`go/cmd/fitness-check-<name>/README.md`):

- Pure logic: node-version, gitignore-why, changelog, changelog-updated, semantic-commit, commit-attribution, read-repo-first, markdown-filename-kebab-case, markdown-filename-camel-case, markdown-front-matter, markdown-no-bold-italic, no-eslint-disable, build-output-untracked, repeated-string-literals
- Parsers and network: the five mermaid diagram/callout checks, vitest-coverage-exclude, dependency-currency (native npm-registry client)
- Native engines: cspell (embedded dictionaries, ~217k words) and jscpd (token-based clone detection) — no external tool needed
- Tool wrappers: prettier, eslint, vitest-coverage-full, swiftlint — these exec the real tool, resolved from `node_modules/.bin` (walking up) then PATH, never npx; a missing binary fails with a one-line install hint

Default run order lives in the runner ([go/cmd/fitness/main.go](go/cmd/fitness/main.go)); opt-in checks (swiftlint, commit-attribution, the mermaid family, and others) are enabled per repo via `.fitnessrc.json`.

## Shared configs

[packages/shared](packages/shared) publishes `@mayjournal/fitness-shared` — tool configs consumed as data. The eslint and prettier checks fall back to these when a repo has no local config; install the package if you want to wire the tools directly:

- `@mayjournal/fitness-shared/eslint.config` – ESLint flat config
- `@mayjournal/fitness-shared/prettier.config` – Prettier (semi, singleQuote, tabWidth 2, trailingComma es5, printWidth 100, sort-json for JSON keys)
- `@mayjournal/fitness-shared/vitest.config` – Vitest coverage thresholds
- `@mayjournal/fitness-shared/cspell` – cspell.json (also the source of the walker's skip dirs and the spell check's project dictionary)

## Development

```bash
npm install        # dev tooling: eslint/prettier stacks, cspell dictionaries for regeneration
npm run build:go   # compile runner + all check binaries into go/bin
npm run fitness    # run the suite on this repo
npm test           # go test ./...
npm run test:scripts  # node --test for the repo scripts
npm run lint
npm run format
```

The spell-check dictionaries under [go/internal/spell/dict](go/internal/spell/dict) are generated from the installed `@cspell` packages — regenerate with `node go/internal/spell/dict/generate.mjs`.

`npm install` runs `prepare`, which points Git at this repo's `githooks/`. If hooks aren't firing — e.g. `core.hooksPath` got reset — re-run:

```bash
git config core.hooksPath githooks
```

See [architecture-index.md](./architecture-index.md) for the C4 model ([architecture/](architecture/)).

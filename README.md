---
# Top-level project config
relatedConfigurations: ['.fitnessrc.json']
---

# fitness

Zero-dependency Go fitness runner that runs checks for local dev, CI/CD, and GenAI workflows to stay aligned with your intended rules and quality bar.

Every check is its own static binary (`fitness-check-<name>`), orchestrated by a `fitness` runner binary. No runtime dependencies, no build step for consumers, instant startup, parallel execution. The full 21-check suite this repo gates its own commits on runs in about 100 milliseconds. The original TypeScript implementation has been retired. Its checks were ported one at a time with side-by-side parity before removal (see the Go-rewrite plan, issue #51).

## Install

<!-- cspell:ignore xzf -->

With the Go toolchain (recommended — works today):

```bash
go install github.com/may-journal/fitness-runner/go/cmd/...@latest
```

One command installs the runner and every check binary into `$HOME/go/bin` — make sure that directory is on PATH. Pin a version with `@v0.20260719.852`. Upgrade by running the same command again with `@latest`.

Prebuilt binaries (no toolchain needed) — grab the tarball for your platform from [the releases page](https://github.com/may-journal/fitness-runner/releases), then extract it onto PATH:

```bash
curl -L -o fitness.tar.gz \
  https://github.com/may-journal/fitness-runner/releases/download/go/v0.20260719.852/fitness-0.20260719.852-darwin-arm64.tar.gz
tar -xzf fitness.tar.gz && mv fitness-*/fitness* ~/bin/
```

Platforms: `darwin-arm64`, `darwin-amd64`, `linux-arm64`, `linux-amd64`. Verify downloads against `checksums.txt` on the same release. Upgrade by grabbing the next release.

From source (contributors):

```bash
git clone https://github.com/may-journal/fitness-runner && cd fitness-runner
cd go && mkdir -p bin && go build -o bin ./cmd/...
```

Put `go/bin` on PATH (or copy the binaries somewhere on it). The runner finds check binaries beside itself first, then on PATH. The Go toolchain is the entire build requirement — no npm, no node.

## Distribution

Every channel delivers the same static binaries: the runner, the changelog stamper, and all 31 checks.

1. `go install` — compiles from source via the public Go module proxy. No artifacts involved, and every published version is cached immutably.
2. GitHub Releases — per-platform tarballs with a checksums file, built and uploaded by CI on every version tag.
3. Homebrew tap — planned, tracked in [issue #47](https://github.com/may-journal/fitness-runner/issues/47). The release-side automation already ships; the tap repo and formula come next.

Versioning: the CHANGELOG timestamp is the only version, and release tags are derived from it. Heading `### 2026.07.19.0837` becomes tag `go/v0.20260719.837` — major pinned at 0 (Go reserves majors of 2 and up for `/vN` module paths), minor is the date, patch is the minute, ordering preserved. `.github/scripts/release-tag.sh` prints the tag for the newest heading, and CI refuses any tag that does not match. Consumers reference the plain version (`@v0.20260719.837`) or `@latest`. The Go proxy caches every published version immutably.

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

The runner invokes each check as `fitness-check-<name> --root <dir> [args…]` with cwd set to the repo root. Context arrives in `FITNESS_*` environment variables: staged files, enabled check names, and the commit message for the commit checks.

- stdout — one JSON result object: `{"ok": bool, "errors": [".."], "filesChecked": n}`
- stderr — human display output (banners, tool passthrough)
- exit code — 0 when the check ran and passed, 1 ran and failed, other values mean it crashed
- `--describe` — prints check metadata (name, timeout budget, context-inline arg) so the runner needs no registry

The runner executes checks in a bounded parallel pool with per-check timeouts. A timeout kills the whole process group, so a hung check's child tree dies with it. Results render as a summary table, and the run exits 1 when any check fails.

## Git hooks

This is the point of the runner: checks enforced automatically on `git commit`. This repo's own hooks live under [githooks/](githooks/). Pre-commit runs the full suite, and commit-msg validates the message through the `semantic-commit` check:

```bash
go/bin/fitness --check=semantic-commit --message="$(cat "$1")"
```

Checks that consume the commit message declare a context-inline argument in their `--describe` metadata. The runner extracts `--message` from single-check argv into the environment.

## GitHub Actions

Some checks also run as GitHub Actions workflows, extending the same rules to targets that are not files in the tree — for example validating plan Issues, which no longer live as files. See [.github/workflows/README.md](.github/workflows/README.md).

## Usage

```bash
go/bin/fitness               # full configured suite
go/bin/fitness prettier      # one check by name
go/bin/fitness --check=eslint
go/bin/fitness prettier --write .   # passthrough args reach the check
```

## Checks

All 31 check names, one binary each under [go/cmd/](go/cmd/), with each check's rule documented in its own README (`go/cmd/fitness-check-<name>/README.md`):

- Pure logic: `node-version`, `gitignore-why`, `changelog`, `changelog-updated`, `changelog-bullets`, `semantic-commit`, `commit-attribution`, `read-repo-first`, `markdown-filename-kebab-case`, `markdown-filename-camel-case`, `markdown-front-matter`, `markdown-links`, `markdown-no-bold-italic`, `no-eslint-disable`, `build-output-untracked`, `repeated-string-literals`, `text-readability`
- Parsers and network: the five mermaid diagram/callout checks, `vitest-coverage-exclude`, `dependency-currency` (native npm-registry client)
- Native engines: `cspell` (embedded dictionaries, ~217k words), `jscpd` (token-based clone detection), and `go-complexity` (cyclomatic complexity ceiling for Go, the house eslint rule's counterpart) — no external tool needed
- Tool wrappers: `prettier`, `eslint`, `vitest-coverage-full`, `swiftlint` — these exec the real tool, resolved from `node_modules/.bin` (walking up) then PATH, never npx. A missing binary fails with a one-line install hint.

Default run order lives in the runner ([go/cmd/fitness/main.go](go/cmd/fitness/main.go)). Opt-in checks (`swiftlint`, `commit-attribution`, the mermaid family, and others) are enabled per repo via `.fitnessrc.json`.

## Shared configs

The opinionated tool configs (eslint flat config, prettier, vitest thresholds, cspell) are embedded inside the check binaries ([go/internal/sharedconf](go/internal/sharedconf)). They materialize to a cache directory on demand. When a check needs a config, the repo's own config file wins. Next comes an installed `@mayjournal/fitness-shared` npm package (previously published versions keep working). The embedded copy is the final fallback. The eslint config references plugins that must exist in the consumer repo — exactly the check's peer-tool contract.

## Development

```bash
cd go
mkdir -p bin && go build -o bin ./cmd/...   # compile runner + all check binaries
go test ./...
cd .. && go/bin/fitness     # run the suite on this repo
```

One-time setup after cloning — point Git at this repo's hooks. Pre-commit restamps a staged CHANGELOG entry and runs the suite. Commit-msg validates through the `semantic-commit` check:

```bash
git config core.hooksPath githooks
```

The spell-check dictionaries under [go/internal/spell/dict](go/internal/spell/dict) are frozen, committed data (provenance in each file header). A Go regeneration tool that fetches dictionary sources directly is deferred to a later milestone.

See [docs/architecture-index.md](./docs/architecture-index.md) for the C4 model ([docs/architecture/](docs/architecture/)).

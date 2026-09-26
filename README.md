---
# Top-level project config
relatedConfigurations: ['.fitnessrc.json']
---

# fitness

Zero-dependency Go fitness runner. It runs checks for local dev, CI/CD, and GenAI workflows to keep code aligned with your rules and quality bar.

Every check is its own static binary (`fitness-check-<name>`), orchestrated by a `fitness` runner binary. No runtime dependencies, no build step for consumers, instant startup, parallel execution. The full suite this repo gates its own commits on runs in about 100 milliseconds. The original TypeScript implementation has been retired, its checks ported one at a time with side-by-side parity (see the Go-rewrite plan, issue #51).

## Install

<!-- cspell:ignore xzf -->

### Go toolchain

Recommended:

```bash
go install github.com/may-journal/fitness-runner/go/cmd/...@latest
```

One command installs the runner and every check binary into `$HOME/go/bin`; put that directory on PATH. Pin a version with `@v0.20260719.852`. Upgrade by rerunning with `@latest`.

### Prebuilt binaries

No toolchain needed. Grab the tarball for your platform from [the releases page](https://github.com/may-journal/fitness-runner/releases), then extract it onto PATH:

```bash
curl -L -o fitness.tar.gz \
  https://github.com/may-journal/fitness-runner/releases/download/go/v0.20260719.852/fitness-0.20260719.852-darwin-arm64.tar.gz
tar -xzf fitness.tar.gz && mv fitness-*/fitness* ~/bin/
```

Platforms: `darwin-arm64`, `darwin-amd64`, `linux-arm64`, `linux-amd64`. Verify downloads against `checksums.txt`. Upgrade by grabbing the next release.

### From source

For contributors:

```bash
git clone https://github.com/may-journal/fitness-runner && cd fitness-runner
cd go && mkdir -p bin && go build -o bin ./cmd/...
```

Put `go/bin` on PATH, or copy the binaries onto it. The runner finds check binaries beside itself first, then on PATH. The Go toolchain is the only build requirement — no npm, no node.

## Usage

```bash
go/bin/fitness               # full configured suite
go/bin/fitness prettier      # one check by name
go/bin/fitness --check=eslint
go/bin/fitness prettier --write .   # passthrough args reach the check
```

## Git hooks

Checks run automatically on `git` operations. This repo's hooks live under [githooks/](githooks/). Point Git at them once:

```bash
git config core.hooksPath githooks
```

- pre-commit: builds, runs `make check`, restamps a staged `CHANGELOG` entry, then runs the full suite — including `no-plans-dir`, which blocks a returning `docs/plans/` file.
- commit-msg: validates the message through `semantic-commit` and the optional `Plan #NN` trailer through `plan-trailer`.
- pre-push: blocks the push unless the pushed commits trace to an approved `Plan` Issue, resolved via `gh`. Chore and docs-only pushes are exempt.

## GitHub Actions

Some checks also run as GitHub Actions workflows. They extend the same rules to non-file targets like plan Issues and pull request descriptions. See [.github/workflows/README.md](.github/workflows/README.md).

## Documentation

- [Distribution and versioning](docs/distribution.md)
- [Check protocol and config](docs/check-protocol.md)
- [Checks catalog and shared configs](docs/checks.md)
- [Development](docs/development.md)
- [Architecture index](docs/architecture-index.md)

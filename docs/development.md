---
relatedConfigurations: ['../.fitnessrc.json']
---

# Development

```bash
cd go
mkdir -p bin && go build -o bin ./cmd/...   # compile fitness, which holds every check
go test ./...
cd .. && go/bin/fitness     # run the suite on this repo
```

One-time setup after cloning — point Git at this repo's hooks. Pre-commit builds the toolchain and runs the suite. Commit-msg validates through the `semantic-commit` check:

```bash
git config core.hooksPath githooks
```

The spell-check dictionaries under [go/internal/spell/dict](../go/internal/spell/dict) are frozen, committed data (provenance in each file header). A Go regeneration tool that fetches dictionary sources directly is deferred to a later milestone.

See [docs/architecture-index.md](architecture-index.md) for the C4 model ([docs/architecture/](architecture/)). Every check is a package under `go/cmd/fitness-check-<name>`, built into `fitness`. A new check folder or doc comment joins it once `go -C go generate ./cmd/fitness` rewrites the generated files: `checks.go`, `topics.go`, `usage.go` and the root `llms.txt`. The local [generated check](../checks/README.md) fails until then.

## Requirement tests

`make requirements` builds the candidate for this host's platform only, then runs the requirement tests against it. Release and CI builds keep all four platforms.

CI splits the tests over two `requirements` jobs per platform with [tests/requirements-shard.sh](../tests/requirements-shard.sh). `sh tests/requirements-shard.sh 1 2` runs the first half against the candidate in `FITNESS_INSTALL`.

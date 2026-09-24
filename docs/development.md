---
relatedConfigurations: ['../.fitnessrc.json']
---

# Development

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

The spell-check dictionaries under [go/internal/spell/dict](../go/internal/spell/dict) are frozen, committed data (provenance in each file header). A Go regeneration tool that fetches dictionary sources directly is deferred to a later milestone.

See [docs/architecture-index.md](architecture-index.md) for the C4 model ([docs/architecture/](architecture/)).

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

See [docs/architecture-index.md](architecture-index.md) for the C4 model ([docs/architecture/](architecture/)). Every check is a package under `go/cmd/fitness-check-<name>`, built into `fitness`.

## Generated docs

One command writes every file generated from the code: `go -C go generate ./cmd/fitness`. It writes the check registry, topics and command line into `fitness`, and each check's README from its package doc comment and config keys. It also writes the check index in [docs/checks.md](checks.md#check-index), the root `llms.txt`, and the README's command list.

Edit the doc comment, usually the check's `doc.go`, never the README. A doc comment link to this repository uses its full GitHub URL, and the README links it relatively. The local [generated check](../checks/README.md) fails a generated file that is stale or edited by hand. Each subcommand needs a `//fitness:usage` line in its comment, which `fitness --help` prints, or generation fails.

## Requirement tests

`make requirements` builds the candidate for this host's platform only, then runs the requirement tests against it. Release and CI builds keep all four platforms.

CI splits the tests over two `requirements` jobs per platform with [tests/requirements-shard.sh](../tests/requirements-shard.sh). `sh tests/requirements-shard.sh 1 2` runs the first half against the candidate in `FITNESS_INSTALL`.

## Skills

Each skill is a folder under [skills](../skills) with a `SKILL.md`, named for its topic, as [ADR 0011](architecture/adr/0011-skills-ship-in-the-bundle.md) decides. The [fitness skill](../skills/fitness/SKILL.md) teaches how to read any failing check, so no check has a skill of its own. Release bundles carry every skill, and `fitness-install --link-skills <folder>` links each into an agent's skills folder; run it again after an upgrade. A skill links the docs that hold the rules and copies none of them.

# ADR 0006 check

Keeps [ADR 0006](../../docs/architecture/adr/0006-biome-stands-in-for-eslint-and-prettier.md): the eslint and prettier checks read Biome through one package.

- Reads every Go source outside tests and docs.
- Fails one outside `go/internal/lang/node/biome` that passes Biome its GitHub reporter flag.
- Fix: call the Biome package, which owns the reporter and its findings.

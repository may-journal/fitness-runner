# 0006 — Biome stands in for ESLint and Prettier

The eslint and prettier checks read Biome through one package.

https://github.com/may-journal/fitness-runner/issues/858

## Check

- Reads every Go source outside tests and docs.
- Fails one outside `go/internal/lang/node/biome` that passes Biome its GitHub reporter flag.
- Fix: call the Biome package, which owns the reporter and its findings.

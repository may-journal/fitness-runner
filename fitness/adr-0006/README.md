# 0006 — Biome stands in for ESLint and Prettier

The eslint and prettier checks read Biome through one package.

## Context

A TypeScript repo that lints and formats with Biome failed the eslint and prettier checks for missing tools it does not want. Skipping both checks would leave its code with no lint or format check.

## Decision

1. A repo chooses Biome with a `biome.json` or `biome.jsonc` at its root and `@biomejs/biome` in its `package.json` dependencies. (KISS)
2. The eslint check then runs `biome lint`, and the prettier check `biome format`, on the paths each would hand its own tool.
3. Both read Biome's GitHub reporter, one finding per line, through `go/internal/biome`.
4. Each check keeps its name, its scope rules, and its `.prettierignore` failure; only the tool changes.
5. Biome 1.8 is the oldest supported, the first to take `--reporter=github`. The version comes from the installed binary, not `package.json`.

## Consequences

1. A Biome repo passes without eslint or Prettier, and still fails on every lint finding and unformatted file.
2. Biome's own config decides its rules, as a repo's own ESLint or Prettier config does.
3. A repo with a Biome config but no `@biomejs/biome` dependency keeps ESLint and Prettier.
4. Tests replay Biome's saved output, like the other peer tools.
5. A repo on Biome before 1.8 fails both checks, naming the version found, until it upgrades.

## Check

- Reads every Go source outside tests and docs.
- Fails one outside `go/internal/lang/node/biome` that passes Biome its GitHub reporter flag.
- Fix: call the Biome package, which owns the reporter and its findings.

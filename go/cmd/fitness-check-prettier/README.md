---
fitnessFunctions: ['main.go']
relatedConfigurations: ['../../internal/sharedconf/config/prettier.config.cjs']
---

# prettier

Runs [Prettier](https://prettier.io) `--check` to ensure files are formatted. Uses a local Prettier config when present (e.g. `.prettierrc.json`, `prettier.config.cjs`, or `package.json` `"prettier"` field). Otherwise it falls back to the shared `prettier.config.cjs`. That comes from an installed `@mayjournal/fitness-shared` package when present, else the copy embedded in the check binary, materialized on demand.

Prettier itself is a peer tool: the check execs the real binary, never npx. It resolves from `node_modules/.bin` (walking up from the repo root) then PATH. It fails with a one-line install hint when the binary is missing.

Run `fitness prettier` from your app root to check formatting. Pass through args to Prettier: `fitness prettier --write .` or `fitness prettier --write src/`. The runner forwards args after the check name to the check, which runs at the repo root.

## Behavior

- Pass: All checked files use Prettier code style.
- In a scoped run: Only changed paths are checked, passing with 0 files when Prettier parses none.
- Otherwise: Runs `prettier --check` on every tracked file from the repo root.
- Config: Local project config if present; otherwise the shared prettier config resolved as above (no copy required in the consumer repo).
- Plugins: the shared config loads `prettier-plugin-packagejson` and `prettier-plugin-sort-json`, so the repo installs both.
- Synced files: the shared config keeps `.github/ISSUE_TEMPLATE/**/*.md` and `.github/fitness-setup.md` at Prettier defaults with an [override](https://prettier.io/docs/configuration#configuration-overrides). Those files arrive from may-journal/.github in that style.
- The check puts the repo's `node_modules` folders on `NODE_PATH`, where the shared config finds them.
- When Prettier fails without naming a file, its own `[error]` lines follow the fallback message, such as a plugin it cannot load.

Errors list file paths that need formatting.

## Biome

A repo that chose Biome formats with it instead, as [ADR 0006](../../../docs/architecture/adr/0006-biome-stands-in-for-eslint-and-prettier.md) records. Such a repo has a `biome.json` or `biome.jsonc` at its root and `@biomejs/biome` in its `package.json` dependencies. The check runs `biome format --reporter=github` on the same paths, and needs no Prettier.

Biome resolves like Prettier, and a missing binary fails with an install hint. Biome before 1.8 fails naming its version, since its CLI rejects the GitHub reporter. Each file Biome would reformat fails by name.

Passthrough args are Prettier's, so the check translates them. Paths such as `.` replace the checked paths, and `--write` writes in place. `--check`, `--list-different`, and `--ignore-unknown` already hold, so they drop. Any other flag fails by name, since Biome would read it as a path.

## Every tracked file

- `--ignore-unknown` lets a file type Prettier cannot parse pass, instead of skipping it by name.
- `--ignore-path /dev/null` and `--with-node-modules` override `.gitignore` and the `node_modules` default.
- A `.prettierignore` at the root fails, even in a repo without `package.json`; delete it and format the files instead.

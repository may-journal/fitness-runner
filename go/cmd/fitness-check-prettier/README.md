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
- Otherwise: Runs `prettier --check --ignore-unknown .` from repo root.
- A file type Prettier has no parser for passes, through `--ignore-unknown`.
- Config: Local project config if present; otherwise the shared prettier config resolved as above (no copy required in the consumer repo).
- Plugins: the shared config loads `prettier-plugin-packagejson` and `prettier-plugin-sort-json`, so the repo installs both.
- The check puts the repo's `node_modules` folders on `NODE_PATH`, where the shared config finds them.
- When Prettier fails without naming a file, its own `[error]` lines follow the fallback message, such as a plugin it cannot load.

Errors list file paths that need formatting.

---
fitnessFunctions: ['main.go']
relatedConfigurations: ['../../internal/sharedconf/config/prettier.config.cjs']
---

# prettier

Runs [Prettier](https://prettier.io) `--check` to ensure files are formatted. Uses a local Prettier config when present (e.g. `.prettierrc.json`, `prettier.config.cjs`, or `package.json` `"prettier"` field). Otherwise it falls back to the shared `prettier.config.cjs` — an installed `@mayjournal/fitness-shared` package when present, else the copy embedded in the check binary, materialized on demand.

Prettier itself is a peer tool: the check execs the real binary, resolved from `node_modules/.bin` (walking up from the repo root) then PATH — never npx. It fails with a one-line install hint when the binary is missing.

Run `fitness prettier` from your app root to check formatting. Pass through args to Prettier: `fitness prettier --write .` or `fitness prettier --write src/`. The runner forwards args after the check name to the check, which runs at the repo root.

## Behavior

- Pass: All checked files use Prettier code style.
- When context has staged files: Only those paths are checked.
- Otherwise: Runs `prettier --check .` from repo root.
- Config: Local project config if present; otherwise the shared prettier config resolved as above (no copy required in the consumer repo).

Errors list file paths that need formatting.

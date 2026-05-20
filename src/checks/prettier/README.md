---
fitnessFunctions: ['prettier']
relatedConfigurations: ['../../../../packages/shared/config/prettier.config.cjs']
---

# prettier

Runs [Prettier](https://prettier.io) `--check` to ensure files are formatted. Uses a local Prettier config when present (e.g. `.prettierrc.json`, `prettier.config.cjs`, or `package.json` `"prettier"` field); otherwise uses this package's `prettier.config.cjs`.

Run `npx fitness prettier` from your app root to check formatting. Pass through args to Prettier: `npx fitness prettier --write .` or `npx fitness prettier --write src/`. The runner layer extracts args after the check name and passes them in context; the check runs in `process.cwd()` (the app using @mayjournal/fitness).

## Behavior

- Pass: All checked files use Prettier code style.
- When context has staged files: Only those paths are checked.
- Otherwise: Runs `prettier --check .` from repo root.
- Config: Local project config if present; otherwise `@mayjournal/fitness` prettier config (no copy required in the consumer repo).

Errors list file paths that need formatting.

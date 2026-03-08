---
fitnessFunctions: ['prettier']
relatedConfigurations: ['../../../prettier.config.cjs']
---

# prettier

Runs [Prettier](https://prettier.io) `--check` to ensure files are formatted. Uses the project's Prettier config (e.g. `.prettierrc.json`, `prettier.config.cjs`, or `package.json` `"prettier"` field).

Run `npx fitness prettier` from your app root to check formatting. Pass through args to Prettier: `npx fitness prettier --write .` or `npx fitness prettier --write src/`. The runner layer extracts args after the check name and passes them in context; the check runs in `process.cwd()` (the app using @mayjournal/fitness).

## Behavior

- Skip: No Prettier config file or `package.json` `"prettier"` field in repo root.
- Pass: All checked files use Prettier code style.
- When context has staged files: Only those paths are checked.
- Otherwise: Runs `prettier --check .` from repo root.

Errors list file paths that need formatting.

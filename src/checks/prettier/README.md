---
fitnessFunctions: ['prettier']
relatedConfigurations: ['../../../prettier.config.cjs']
---

# prettier

Runs [Prettier](https://prettier.io) `--check` to ensure files are formatted. Uses the project's Prettier config (e.g. `.prettierrc.json`, `prettier.config.cjs`, or `package.json` `"prettier"` field).

## Behavior

- Skip: No Prettier config file or `package.json` `"prettier"` field in repo root.
- Pass: All checked files use Prettier code style.
- When context has staged files: Only those paths are checked.
- Otherwise: Runs `prettier --check .` from repo root.

Errors list file paths that need formatting.

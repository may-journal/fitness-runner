---
fitnessFunctions: ['prettier']
relatedConfigurations: ['../../../prettier.config.cjs']
---

# prettier

Runs [Prettier](https://prettier.io) `--check` to ensure files are formatted. Uses the project's Prettier config (e.g. `.prettierrc.json` or `prettier.config.cjs`).

## Behavior

- Skip: No Prettier config found in repo root.
- Pass: All checked files use Prettier code style.
- When context has staged files: Only those paths are checked.
- Otherwise: Runs `prettier --check .` from repo root.

Errors list file paths that need formatting.

---
relatedConfigurations: ['../../package.json']
---

# nvm-use

Loads [nvm](https://github.com/nvm-sh/nvm) and runs `nvm use` for the Node version in the repo root `.nvmrc`.

Used by Git hooks in `githooks/` and package `ci` scripts so hooks and workspace scripts use the expected Node version.

## Usage

From repo root:

```bash
bash scripts/nvm-use/nvm-use.sh
```

Git hooks (`githooks/`):

- `githooks/pre-commit`
- `githooks/commit-msg`

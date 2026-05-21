---
relatedConfigurations: ['../../package.json']
---

# ensure-changelog-timestamp

Pre-commit helper: when `CHANGELOG.md` is staged, bumps the latest `### YYYY.MM.DD.HHMM` section heading and every workspace `package.json` version suffix to the current timestamp, then runs `npm install`, Prettier, and re-stages touched files.

## When it runs

`githooks/pre-commit` (after `nvm-use`), before fitness and lint.

## Behavior

1. No-op if `CHANGELOG.md` is not in the staged file list.
2. Rewrites the first changelog section heading timestamp.
3. Updates `version` in root and all `packages/**/package.json` files that use the `0.1.0-YYYY.MM.DD.HHMM` pattern.
4. Refreshes `package-lock.json` and formats changed files.

## Usage

Normally invoked by the pre-commit hook only. Manual run from repo root:

```bash
node scripts/ensure-changelog-timestamp/index.cjs
```

Requires staged `CHANGELOG.md` to do work.

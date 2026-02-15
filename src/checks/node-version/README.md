---
relatedConfigurations: ["../../../package.json"]
---

# node-version

Validates that the current Node version satisfies the repo’s `.nvmrc` (compare only; run `nvm use` yourself if needed).

## Behavior

- **Pass:** `.nvmrc` exists and its major version is ≤ current Node’s major (e.g. `.nvmrc` has `24` and Node is v24.x or higher).
- **Fail:** `.nvmrc` missing → `missing .nvmrc`.
- **Fail:** Node too old → `Node vX.Y.Z does not satisfy .nvmrc (requires N.x). Run: nvm use`.

Accepts plain or `v`-prefixed versions in `.nvmrc` (e.g. `24` or `v24`).

## Contributing

This README is the canonical description for this check; `.cursor/rules/node-version.mdc` points Cursor here. This check is a self-contained sub-project. To support other version files or engines, extend the check and tests here and update the README.

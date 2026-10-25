---
name: dependency-currency
description: Fix a failing dependency-currency check on npm dependencies. Use when a declared dependency is behind its latest published version.
---

# Update npm dependencies

Every dependency declared in `package.json` is at its latest published version. [The check README](../../README.md) explains which manifests it reads and why `latest` is the target.

## Read the failure

- `Dependencies behind their latest published version — update them (npm install <pkg>@latest) or pin intentionally:` starts the list, then each line reads `left-pad: 1.2.0 → 1.3.0`. Run `npm install left-pad@latest`, then run the tests.
- A line reading `left-pad: missing → 1.3.0` means the dependency is declared but not installed. Run `npm install`.

If the update breaks the code, fix the code for the new version.

## Check it first

Run `fitness dependency-currency` from the repo root. Exit 0 means every dependency is current. Without network access the check passes, so run it online.

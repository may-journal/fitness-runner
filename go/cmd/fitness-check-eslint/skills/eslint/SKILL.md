---
name: eslint
description: Fix a failing eslint check on tracked TypeScript and JavaScript files. Use when the check reports lint errors, a missing eslint, or an ignored file.
---

# Fix ESLint findings

Every lintable file must pass the shared ESLint config with no errors. [The check README](../../README.md) lists the files it lints and how the config resolves.

## Read the failure

- `path:line:col - message (ruleId)`: ESLint's own finding. Change the code to satisfy the named rule, as in sorting object keys for `sort-keys`.
- `path - ESLint ignores this tracked file; remove the ignore pattern so it is linted`: delete the ignore pattern that matches the file.
- `eslint not found in node_modules/.bin (walking up from the repo root) or on PATH. Run: npm install --save-dev eslint`: install eslint and the plugins the shared config imports.
- `ESLint reported issues. Run: npx eslint .`: the output named no file. Run eslint yourself to see the cause.

A repo that lints with Biome gets the same line format with `(rule)` in place of the rule id. See the README section on Biome.

## Check it first

Run `fitness eslint` from the repo root. Exit 0 means it passes.

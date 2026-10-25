---
name: node-version
description: Fix a failing node-version check on the Node runtime. Use when .nvmrc is missing or Node is older than it requires.
---

# Match Node to .nvmrc

The Node major version you run is at least the one in `.nvmrc`, written as `24` or `v24`. [The check README](../../README.md) states the rule.

## Read the failure

- `missing .nvmrc`: add a `.nvmrc` at the repo root holding the major version the project needs.
- `Node v22.1.0 does not satisfy .nvmrc (requires 24.x). Run: nvm use`: switch to the required version with `nvm use`, then rerun.
- `node not found on PATH. Install Node.js (e.g. via nvm) to run the node-version check`: install Node.

The check only compares versions. It never switches Node for you.

## Check it first

Run `fitness node-version` from the repo root. Exit 0 means the versions match.

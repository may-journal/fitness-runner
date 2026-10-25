---
name: jscpd
description: Fix a failing jscpd check on duplicated code. Use when duplicated lines pass the threshold.
---

# Remove duplicated code

Duplicated lines stay under 1% of the codebase, counting blocks of 5 lines and 50 tokens. [The check README](../../README.md) lists what is scanned and what is left out.

## Read the failure

- `ERROR: jscpd found too many duplicates (1.4%) over threshold (1.0%)`: the share of duplicated lines is too high.
- `path: 24 duplicated lines`: one line for each file holding duplicates. Open the file and find the block that repeats elsewhere.

Extract the repeated block into one shared function, constant or helper that both places call. In tests, share setup in a helper and keep each case's assertions in its own test. Markdown and JSON count too, so link to a doc instead of copying it.

## Check it first

Run `fitness jscpd` from the repo root. Exit 0 means the duplicate share is under the threshold.

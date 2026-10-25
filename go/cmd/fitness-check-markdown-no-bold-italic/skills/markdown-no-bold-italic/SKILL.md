---
name: markdown-no-bold-italic
description: Fix a failing markdown-no-bold-italic check on a markdown file. Use when a doc is refused for bold or italic emphasis.
---

# Remove bold and italic emphasis

Markdown prose uses no bold or italic. [The check README](../../README.md) lists what is ignored: fenced code, inline code and link blocks.

## Read the failure

- `docs/a.md: disallowed **bold** (use only when explicitly required): "**Note**"`: drop the asterisks.
- `docs/a.md: disallowed __bold__ (use only when explicitly required): "__Note__"`: drop the underscores.
- `docs/a.md: disallowed *italic* (use only when explicitly required): "*why*"`: drop the asterisks.
- `docs/a.md: disallowed _italic_ (use only when explicitly required): "_why_"`: drop the underscores.

Put names and identifiers in backticks. Use a heading for a label. Carry stress in word choice.

## Check it first

Run `fitness markdown-no-bold-italic` from the repo root. Exit 0 means no emphasis is left.

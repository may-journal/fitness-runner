---
name: mermaid-diagram-table-gap
description: Fix a failing mermaid-diagram-table-gap check on a doc. Use when prose sits between a diagram and its callout table.
---

# Close the gap between diagram and table

The layout is diagram, legend, one caption line, then the callout table. [The check README](../../README.md) states the rule, and the [containers view](../../../../../docs/architecture/02-containers.md) shows the shape.

## Read the failure

- `docs/a.md: prose between the diagram and its callout table (line 14) — move detail into the callout table`: move the sentence into a table row, or delete it if the table already says it.

The one allowed line opens with `Numbers` and says they `match the callout table`. Prose before the diagram or after the table is never judged.

## Check it first

Run `fitness mermaid-diagram-table-gap` from the repo root. Exit 0 means only blank lines and the caption sit in the gap.

---
name: mermaid-callouts
description: Fix a failing mermaid-callouts check on a diagram and its callout table. Use when callout numbers and table rows do not match one to one.
---

# Match diagram callouts to table rows

Each callout number in a numbered diagram has exactly one row in the callout table after it. Each row has a number in the diagram. [The check README](../../README.md) defines the pairing. See the [containers view](../../../../../docs/architecture/02-containers.md) for the shape.

## Read the failure

- `docs/a.md: diagram callout 4 has no matching table row (line 10)`: add row 4, or remove the number from the diagram.
- `docs/a.md: callout table row 5 has no matching diagram callout (line 20)`: add the number to the diagram, or delete the row.
- `docs/a.md: diagram callout 2 appears 2 times (line 10)`: give each element its own number. The same message exists for table rows.
- `docs/a.md: mermaid diagram at line 10 has numbered callouts but no associated callout table`: add the table right after the diagram and its legend.
- `docs/a.md: callout table at line 20 has no preceding mermaid diagram`: add the diagram above it, or remove the table.

## Check it first

Run `fitness mermaid-callouts` from the repo root. Exit 0 means every number pairs with one row.

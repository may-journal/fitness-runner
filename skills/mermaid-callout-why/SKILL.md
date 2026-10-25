---
name: mermaid-callout-why
description: Fix a failing mermaid-callout-why check on a callout table. Use when a callout table lacks a Why column or a row leaves it empty.
---

# Give every callout a Why

Each numbered callout table has a `Why` column, and every numbered row fills it. [The check README](../../go/cmd/fitness-check-mermaid-callout-why/README.md) states the rule. The [containers view](../../docs/architecture/02-containers.md) shows a table with the column.

## Read the failure

- `docs/a.md: callout table at line 12 is missing a "Why" column`: add a `Why` header cell and a reason in each row.
- `docs/a.md: callout table row 3 has an empty "Why" cell (line 12)`: write the reason this element or edge exists. The line number is the table's first line.

Write why the element matters. The diagram and the other columns already say what it is.

## Check it first

Run `fitness mermaid-callout-why` from the repo root. Exit 0 means every table has a filled Why column.

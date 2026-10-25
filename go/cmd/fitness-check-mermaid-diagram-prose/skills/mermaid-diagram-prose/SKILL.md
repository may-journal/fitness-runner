---
name: mermaid-diagram-prose
description: Fix a failing mermaid-diagram-prose check on edge labels. Use when a relationship label holds words beyond its callout number.
---

# Keep descriptions out of the diagram

In a diagram with a callout table, a relationship label is the callout number alone. The words belong in the table. [The check README](../../README.md) states the rule, and the [components view](../../../../../docs/architecture/03-components.md) follows it.

## Read the failure

- `docs/a.md: diagram at line 10 has a relationship label with prose ("6 Uses") — put descriptions in the callout table`: change the label to `6`, and put `Uses` in the table row for 6.

## Check it first

Run `fitness mermaid-diagram-prose` from the repo root. Exit 0 means every relationship label is a bare number.

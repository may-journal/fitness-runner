---
name: mermaid-legend
description: Fix a failing mermaid-legend check on a flowchart. Use when numbered nodes lack a style class or the diagram lacks a classDef legend.
---

# Style numbered nodes and define the classes

Every numbered flowchart node carries a `:::class` assignment, and the diagram declares its classes with `classDef`. [The check README](../../go/cmd/fitness-check-mermaid-legend/README.md) states the rule. The [user journey view](../../docs/architecture/05-user-journey.md) styles its nodes.

## Read the failure

- `docs/a.md: callout 3 (open) has no style class (diagram at line 10)`: add `:::action` or another class to node `open`.
- `docs/a.md: diagram at line 10 has numbered callouts but no classDef legend`: add a `classDef` line for each class you use.

## Check it first

Run `fitness mermaid-legend` from the repo root. Exit 0 means every numbered node is styled and the classes are defined.

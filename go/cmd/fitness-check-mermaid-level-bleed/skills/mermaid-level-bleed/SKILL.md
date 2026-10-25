---
name: mermaid-level-bleed
description: Fix a failing mermaid-level-bleed check on stacked architecture views. Use when a lower level repeats a callout description from the level above.
---

# Say something new at each level

A callout description in a lower view must add detail the upper view lacks. [The check README](../../README.md) states the rule. Compare the [containers view](../../../../../docs/architecture/02-containers.md) with the [components view](../../../../../docs/architecture/03-components.md).

## Read the failure

- `docs/architecture/03-components.md: description "Runs the checks" repeats level 2 verbatim — lower levels should add level-specific rationale`: rewrite the lower row to say what this level adds, such as the component's own job or the reason for its boundary.

## Check it first

Run `fitness mermaid-level-bleed` from the repo root. Exit 0 means no description repeats the level above.

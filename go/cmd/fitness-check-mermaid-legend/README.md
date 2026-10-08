---
relatedConfigurations: ['../../../.fitnessrc.json']
---

# mermaid-legend

Part of the mermaid diagram + callout table fitness-function set
([#28](https://github.com/may-journal/fitness-runner/issues/28)): numbered callouts must have a legend and consistent styling.

## Behavior

Requires every numbered flowchart callout node to carry a `:::class` style assignment, and flags nodes with no styling. Consistent styling keeps callouts legible in a legend.

Runs on every `.md` file with a mermaid diagram or callout table; files with
neither are skipped and don't count toward `filesChecked`.

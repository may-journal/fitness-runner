---
relatedConfigurations: ['../../../.fitnessrc.json']
---

# mermaid-diagram-prose

Part of the mermaid diagram + callout table fitness-function set
([#28](https://github.com/may-journal/fitness-runner/issues/28)): diagram labels must not duplicate callout table prose.

## Behavior

When a diagram is paired with a callout table, it flags edge labels with prose after a callout number or letter. For example, `"6 Uses"` should be `"6"` and `"A calls"` should be `"A"`. Descriptions belong in the table, not the diagram.

Runs on every `.md` file with a mermaid diagram or callout table; files with
neither are skipped and don't count toward `filesChecked`.

---
relatedConfigurations: ['../../../.fitnessrc.json']
---

# mermaid-syntax

Part of the mermaid diagram + callout table fitness-function set ([#28](https://github.com/may-journal/fitness-runner/issues/28)). It fails a diagram that would not render, so a broken diagram never reaches a reader.

## Behavior

The check parses each ` ```mermaid ` block in pure Go, with no npm or browser. Errors name the document line and the diagram's opening fence.

- Flowcharts and graphs: the header, node shapes, edges and their labels, `&` groups, `subgraph` blocks, and styling statements.
- An unquoted flowchart label holding brackets or quotes fails; wrap it in double quotes.
- Sequence diagrams: participants, notes, messages with a target and a colon, and blocks with their branches and `end`.
- Other known types, such as `C4Context` and `classDiagram`: balanced brackets and quotes only, since a full grammar for each is out of reach.
- `erDiagram`: its header only, since its cardinality marks pair no brackets.
- An unknown diagram type fails.

It runs on every `.md` file with a mermaid diagram and, given `--body-file`, on one Plan, PR, or comment body. Files with no diagram are skipped and don't count toward `filesChecked`.

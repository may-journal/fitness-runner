---
relatedConfigurations: ['../../../.fitnessrc.json']
---

# mermaid-diagram-table-gap

Part of the mermaid diagram + callout table fitness-function set
([#28](https://github.com/may-journal/fitness-runner/issues/28)). The callout table is the single home for detail. No loose prose may sit between a numbered diagram (or its legend) and its table ([#66](https://github.com/may-journal/fitness-runner/issues/66)).

Opt-in — not in the runner's default list. Enable it via `.fitnessrc.json`:

```json
{ "checks": ["mermaid-diagram-table-gap"] }
```

## Behavior

The C4 doc layout is diagram → legend → one caption line → callout table, with every description in the table. This flags any non-blank markdown line in the gap between the table and the last mermaid block before it. That block is the legend when present, else the diagram. Such a paragraph just duplicates the table.

The single caption line is allowed: a line that opens with `Numbers` and states they `match the callout table`. Any wording in between and any trailing clause after is fine. Prose before the diagram and prose after the table are both left alone — only the diagram-to-table gap is policed. A sibling check, `mermaid-diagram-prose`, owns the different rule of prose inside the diagram's own labels.

Runs on every `.md` file with a mermaid diagram or callout table; files with
neither are skipped and don't count toward `filesChecked`.

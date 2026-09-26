---
relatedConfigurations: ['../../../.fitnessrc.json']
---

# mermaid-level-bleed

Part of the mermaid diagram + callout table fitness-function set
([#28](https://github.com/may-journal/fitness-runner/issues/28)): stacked C4 levels must not restate the parent.

Opt-in — not in the runner's default list. Enable it via `.fitnessrc.json`:

```json
{ "checks": ["mermaid-level-bleed"] }
```

## Behavior

Across numbered `architecture/NN-*.md` files, compares callout descriptions at adjacent levels. It warns when a lower level repeats an upper level verbatim, adding nothing new.

Runs on every `.md` file with a mermaid diagram or callout table; files with
neither are skipped and don't count toward `filesChecked`.

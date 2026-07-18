---
relatedConfigurations: ['../../../.fitnessrc.json']
---

# mermaid-callout-why

Part of the mermaid diagram + callout table fitness-function set
([#28](https://github.com/may-journal/fitness-runner/issues/28)): callout tables must include a Why column.

Opt-in — not part of the runner's default list. Enable it via `.fitnessrc.json`:

```json
{ "checks": ["mermaid-callout-why"] }
```

## Behavior

For each numbered callout table, requires a `Why` column (case-insensitive) in the header, and warns when a numbered row leaves its `Why` cell empty.

Runs on every `.md` file with a mermaid diagram or callout table; files with
neither are skipped and don't count toward `filesChecked`.

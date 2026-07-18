---
relatedConfigurations: ['../../../package.json']
---

# mermaid-diagram-prose

Part of the mermaid diagram + callout table fitness-function set
([#28](https://github.com/may-journal/fitness-runner/issues/28)): diagram labels must not duplicate callout table prose.

Opt-in — not part of `defaultChecks`. Enable it via `.fitnessrc`:

```ts
export default { checks: ['mermaid-diagram-prose'] };
```

## Behavior

When a diagram is paired with a callout table, flags relationship/edge labels that carry prose beyond a callout number (e.g. `"6 Uses"` instead of `"6"`) — descriptions belong in the table, not the diagram.

Runs on every `.md` file with a mermaid diagram or callout table; files with
neither are skipped and don't count toward `filesChecked`.

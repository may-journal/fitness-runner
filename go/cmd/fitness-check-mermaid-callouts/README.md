---
relatedConfigurations: ['../../../.fitnessrc.json']
---

# mermaid-callouts

Foundational check of the mermaid diagram + callout table fitness-function set
([#28](https://github.com/may-journal/fitness-runner/issues/28)). Ensures every
numbered mermaid diagram has an associated callout table and that their callout
numbers are a 1-1 match.

Opt-in — not part of `defaultChecks`. Enable it via `.fitnessrc`:

```ts
export default { checks: ['mermaid-callouts'] };
```

## Behavior

For each `.md` file, diagrams (` ```mermaid ` blocks) are paired with the
numbered callout table that follows them (a GFM table whose first header cell is
`#`, `No`, `Callout`, or `Ref`).

- Pass: every callout number in the diagram maps to exactly one table row,
  and vice versa. Un-numbered diagrams need no table.
- Fail (orphan callout): `diagram callout N has no matching table row`.
- Fail (orphan row): `callout table row N has no matching diagram callout`.
- Fail (duplicate): `diagram callout N appears K times` / `callout table row N appears K times`.
- Fail (unpaired diagram): `mermaid diagram … has numbered callouts but no associated callout table`.
- Fail (unpaired table): `callout table … has no preceding mermaid diagram`.

Callout numbers are read from the leading integer of quoted diagram labels
(`Rel(a, b, "6")`, `Container(x, "3", …)`) and from the first (`#`) column of the
callout table.

## Scope

Runs on every `.md` file that contains a diagram or callout table. Files with
neither are skipped and don't count toward `filesChecked`.

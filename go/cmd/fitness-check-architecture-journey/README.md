---
fitnessFunctions: ['architecture-journey']
relatedConfigurations: ['../../../.fitnessrc.json']
---

# architecture-journey

Keeps the user journey and the requirement docs covering each other. It reads the shared model from `internal/architecture.Load`. Scoped runs still read the whole journey view and every requirement doc.

## The requirement table

The journey view (`05-user-journey.md` under `architecture/` or `docs/architecture/`) holds a table whose first header cell is `Requirement`. Each row names one requirement doc and the journey nodes that fulfil it:

```markdown
| Requirement                             | Journey                     |
| --------------------------------------- | --------------------------- |
| [0001](../requirements/0001-open.md)    | `open`, `done`              |
| [0002](../requirements/0002-offline.md) | `open`                      |
```

- The requirement cell links the doc, or states its four-digit ID such as `0001`.
- An acceptance such as `0001.1` is not a doc ID. Acceptances keep their one owning test under the `requirements` check.
- The journey cell lists Mermaid node IDs from the journey diagrams, comma-separated.
- IDs are stable, so renaming a label keeps its rows. Renaming an ID exposes the stale row.
- One row may list several nodes, and one node may appear in several rows.

Personas and layout subgraphs promise no behavior and need no row. Legends stay out of the model. Every other journey node, such as a screen, action, or system step, needs a row.

## Failures

Every error names the view line it concerns and the requirement ID.

- A requirement doc with no row, named at the table's header line.
- A row naming an acceptance ID, a doc that does not exist, or no doc ID at all.
- A second row for one requirement, naming the first.
- A row listing no node, or a node the journey diagrams do not declare.
- A behavior node no row lists, named at its diagram line.

A repo without architecture views passes with zero files. A missing journey view belongs to `architecture-contract`.

## Requirements

[Requirement 0071](../../../docs/requirements/0071-checks-architecture-journey.md) has one installed-command test per acceptance.

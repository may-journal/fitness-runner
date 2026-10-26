---
fitnessFunctions: ['architecture-contract']
relatedConfigurations: ['../../../.fitnessrc.json']
---

# architecture-contract

Reads one model from the diagrams and callout tables already in each architecture view. It adds no fields or columns. Later checks share the model through `internal/architecture.Load` and requirement clauses through `internal/requirement.Load`.

## Views

The roots are `architecture/` and `docs/architecture/`. Both roots add to one model.

| File | View |
| --- | --- |
| `01-system-context.md` | People and systems |
| `02-containers.md` | Runtime units |
| `03-components.md` | Components and their sources |
| `04-code.md` | Classes and functions |
| `05-user-journey.md` | Journey steps and the requirement IDs they fulfil |

No views means a pass with zero files, unless the repo has product code in a [language module](../../internal/lang/langs/langs.go)'s sources. Then the check fails and asks for all five. Product code is any such tracked file except tests, tool configs, and files marked `linguist-generated`. Each module names its own tests and tool configs.

Any view requires all five, and each view needs a diagram outside its legends. Scoped runs read the whole model.

## Templates

Each view has a template in [internal/architecture/templates](../../internal/architecture/templates), and fitness ships all five inside the binary. Start each view the repo lacks as a copy of its template:

```bash
mkdir -p docs/architecture
for view in 01-system-context 02-containers 03-components 04-code 05-user-journey; do
  test -e "docs/architecture/$view.md" || curl -fsSL -o "docs/architecture/$view.md" \
    "https://raw.githubusercontent.com/may-journal/fitness-runner/v1/go/internal/architecture/templates/$view.md"
done
```

The `doc-template` check judges every view under `docs/architecture/` or `architecture/` against its shipped template. A repo needs no copy of the templates; see [shipped templates](../fitness-check-doc-template/README.md#shipped-templates). Each template's comments say what to write.

| View | Diagram | Kinds | Palette |
| --- | --- | --- | --- |
| 01 System context | `C4Context` | Person, System, `System_Ext` | System `#1d4ed8` |
| 02 Containers | `C4Container` | Person, `System_Boundary`, Container, `System_Ext` | Container `#0f766e` |
| 03 Components | `C4Component` | Person, `Container_Boundary`, Component, `System_Ext` | Component `#b45309` |
| 04 Code | `classDiagram` | class | class `#6d28d9` |
| 05 User journey | `flowchart TB` | the four journey classes | its `classDef` lines |

People are slate `#334155` and outside systems grey `#64748b` on every level, all with white text. Each fill is a fixed color that keeps 4.5:1 contrast with white, so GitHub's light and dark themes draw it alike.

## What the diagrams already say

| Content | Model fact |
| --- | --- |
| Mermaid ID, such as `user` in `Person(user, "1 User")` | Stable identity; the same ID in two views is one element |
| C4 macro, flowchart `:::class`, or `class` | Element kind |
| Label without its callout number | Name |
| Enclosing C4 boundary, subgraph, or namespace | Parent |
| `Rel(...)` or an arrow | Directed relationship; left-facing class arrows and `Rel_Back` are reversed |
| The arrow itself, such as `-->` or `..>` | Relationship kind, kept for the code check |
| A class body line, such as `+Ok` or `+Run(string) error` | Member of that class |
| A class annotation, such as `<<abstract>>` | Annotation of that class |
| Callout number | Display reference that binds a table row; renumbering keeps identity |

Legends stay out of the model: unnumbered diagrams, C4 diagrams titled `Legend`, and diagrams under a `Legend` heading with their tables.

## What callout rows already say

| Row content | Model fact |
| --- | --- |
| Code span or diagram text naming a path under a tracked top-level folder | Source claim by the element |
| Code span or label naming a tracked root-level source, such as `index.ts` | Source claim by the element |
| A trailing `*` in such a path, as in `go/cmd/fitness-check-*` | One claim on every sibling the pattern matches |
| Link to a tracked source file, optionally `#L3` or `#L3-L9` | Static evidence for the element or edge |
| Link to `docs/requirements/NNNN-*.md`, or a code span such as `0001.1` | Requirement reference |

Other text is prose. For example, a consumer's `node_modules/.bin` is no claim, since `node_modules` is not a tracked folder.

## Failures

- Duplicate or invalid IDs, and edges to IDs the diagram does not declare.
- A container in a container view, or a component in a component view, outside any boundary.
- One person ID with two names, or one person name on two IDs.
- Two elements claiming one source path when neither encloses the other.
- Claimed paths, evidence files, or line ranges that do not exist.
- Requirement references that resolve to no doc or acceptance. Doc shape errors belong to the `requirements` check.
- Unsupported Mermaid forms and statements.

Front matter, accessibility lines, C4 `Update…` style macros, and named C4 arguments such as `$tags` are styling and stay out of the model.

Every error names the view line. An error about two places names both.

## Sources and proof

Every tracked source stays in the inventory as product, test, tool, or generated, with the elements whose claims cover it. Tool sources are scripts and root-level script tool configs such as `rollup.config.mjs`. Unowned sources are listed, not failed; the component check decides ownership.

Static evidence only locates a line; later checks resolve what it proves. Runtime proof has no form in the diagrams, so no edge counts as proved at runtime.

## Requirements

[Requirement 0070](../../../docs/requirements/0070-checks-architecture-contract.md) has one installed-command test per acceptance.

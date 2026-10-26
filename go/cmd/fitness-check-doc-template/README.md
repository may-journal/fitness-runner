---
fitnessFunctions: ['doc-template']
relatedConfigurations: ['../../../.fitnessrc.json']
---

# doc-template

A template file defines a markdown document's shape, and its neighbors must match it. A folder may hold several templates, one per kind of doc. This is the file-level counterpart of `plan-structure` and `pr-structure`, sharing the `mdtemplate` engine.

## What is a template

- `template.md` or `*.template.md` — a template that governs its neighbors.
- Any `*.md` directly in a `templates` subfolder — a template for that subfolder's parent folder.
- A GitHub PR or Issue template under `.github` (`PULL_REQUEST_TEMPLATE.md`, `ISSUE_TEMPLATE/*.md`).

## Rules

- Every `##` section of a template must carry a guiding `<!-- comment -->`.
- A folder's templates are its own `template.md`-style files plus every file in its `templates` subfolder.
- Walking up from each file's own folder, the first folder with a template that governs it decides.
- The file must have the same `##` sections as any one of that folder's templates.
- So `docs/arch/adr/0001-x.md` follows `docs/arch/adr/template.md`, while `docs/arch/04-code.md` follows `docs/arch/templates/*.md`.
- A GitHub template gets the comment rule only; its body is checked by `pr-structure` and `plan-structure`.

## Named templates

A file in a `templates` folder governs alone the neighbor that shares its name, so `docs/architecture/templates/04-code.md` judges `docs/architecture/04-code.md`. It governs no other doc, so the other neighbors match the folder's other templates. A neighbor in a folder of only named templates follows the nearest folder above with others, so `docs/architecture/notes.md` follows `docs/template.md`. A named template holds its namesake to its whole shape:

- Sections: the template's `##` sections open the doc, in order. Sections of the doc's own may follow.
- Legends: a section with a legend in the template needs a legend of the same Mermaid type.
- Legend kinds: a diagram with no callout numbers, one titled `Legend`, or one under a `Legend` heading.
- Diagram types: each diagram and legend in a section has the Mermaid type of the template's diagram or legend there.
- Callout tables: a section with one in the template needs one, and each has the template's columns.
- Captions: the paragraph above each callout table starts with the first sentence of the template's caption.
- Palette: see below, for templates with C4 or class diagrams.

## Palette

A template's C4 and `classDiagram` diagrams, legends included, set its palette: the element kinds they use, and the one style each kind carries. Db and Queue forms count as their base kind, and `Person_Ext` as Person. Every C4 or class diagram in the doc must then:

- Use only those kinds, so a Container in the system context or a `System_Boundary` in the components view fails.
- Give each element of a styled kind exactly its kind's style: an `UpdateElementStyle` line in C4, a `style` line for a class.
- Write that style in any order, spacing and case.
- Leave each unstyled kind, such as a boundary, without style.
- Hold no other style: another `Update…Style` macro, a color argument, `classDef`, `cssClass`, `:::`, or `%%{init}`. `UpdateLayoutConfig` only lays out, so it stays.

The journey template draws a flowchart, so its palette judges nothing.

## Shipped templates

Fitness ships the five architecture view templates inside the binary. A view under `docs/architecture/` or `architecture/` follows its shipped template, such as [04-code.md](../../internal/architecture/templates/04-code.md), with every rule above. A repo's own copy in the views' `templates` folder governs instead, as a named template. The [architecture-contract README](../fitness-check-architecture-contract/README.md#templates) lists each level's form and palette.

## Behavior

- Pass: every template guides each section, and every governed file matches. A repo with no templates passes.
- Fail: a template section lacks a comment, or a governed file matches none of its templates.
- With one template, each difference is its own error. With several, one error names them all.

## Contributing

This README is the canonical description for this check. It is a self-contained binary (`fitness-check-doc-template`). The section comparison lives in `internal/mdtemplate`; extend the check and tests here and keep the README in sync.

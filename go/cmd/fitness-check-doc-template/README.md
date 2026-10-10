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
- Walking up from each file's own folder, the first folder with any template governs it.
- The file must have the same `##` sections as any one of that folder's templates.
- So `docs/arch/adr/0001-x.md` follows `docs/arch/adr/template.md`, while `docs/arch/04-code.md` follows `docs/arch/templates/*.md`.
- A GitHub template gets the comment rule only; its body is checked by `pr-structure` and `plan-structure`.

## Behavior

- Pass: every template guides each section, and every governed file matches. A repo with no templates passes.
- Fail: a template section lacks a comment, or a governed file matches none of its templates.
- Each error lists one difference from the file's closest template, the one with the fewest, and names it when there are several.

## Contributing

This README is the canonical description for this check. It is a self-contained binary (`fitness-check-doc-template`). The section comparison lives in `internal/mdtemplate`; extend the check and tests here and keep the README in sync.

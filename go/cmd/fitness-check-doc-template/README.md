---
fitnessFunctions: ['doc-template']
relatedConfigurations: ['../../../.fitnessrc.json']
---

# doc-template

A template file defines a markdown document's shape, and its neighbors must match it. This is the file-level counterpart of `plan-structure` and `pr-structure`, sharing the `mdtemplate` engine.

## What is a template

- `template.md` or `*.template.md` — a template that governs its neighbors.
- A GitHub PR or Issue template under `.github` (`PULL_REQUEST_TEMPLATE.md`, `ISSUE_TEMPLATE/*.md`).

## Rules

- Every `##` section of a template must carry a guiding `<!-- comment -->`.
- For a `template.md`-style file, each markdown file in its folder or a descendant must have the same `##` sections.
- The nearest template above a file wins.
- A GitHub template gets the comment rule only; its body is checked by `pr-structure` and `plan-structure`.

## Behavior

- Pass: every template guides each section, and every governed file matches. A repo with no templates passes.
- Fail: a template section lacks a comment, or a governed file's sections diverge — one error per violation.

## Contributing

This README is the canonical description for this check. It is a self-contained binary (`fitness-check-doc-template`). The section comparison lives in `internal/mdtemplate`; extend the check and tests here and keep the README in sync.

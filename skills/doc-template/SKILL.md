---
name: doc-template
description: Fix a failing doc-template check on a template or a doc that follows one. Use when a doc's sections do not match its folder's template.
---

# Match a doc to its template

A template defines the `##` sections of its neighbors, and each section carries a guiding comment. [The check README](../../go/cmd/fitness-check-doc-template/README.md) says which template governs which doc.

## Read the failure

- ``docs/adr/template.md: section `## Context` needs a guiding <!-- comment -->``: add a comment under that heading that says what to write.
- `docs/adr/0002-x.md: sections match none of its templates: docs/adr/templates/a.md, docs/adr/templates/b.md`: give the doc the `##` sections of one listed template.
- ``docs/adr/0002-x.md: Missing `## Context` section`` and ``Unexpected `## Notes` section — ...``: when one template governs, add the missing heading or remove the extra one.
- `sections must open with ... as <template> does`: a named template governs the doc. Open it with the template's `##` sections in order.

Fix the doc to fit the template. Do not move the doc out of the folder or delete the template.

## Check it first

Run `fitness doc-template` from the repo root. Exit 0 means every template is guided and every doc matches.

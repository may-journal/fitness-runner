---
name: markdown-filename-camel-case
description: Fix a failing markdown-filename-camel-case check on a markdown file name. Use when a doc is refused because its name is not camelCase.
---

# Name markdown files in camelCase

A markdown basename starts with a lowercase letter and then uses letters and digits only, such as `apiDesign.md`. The rule lives in the [kebab-case README](../../../fitness-check-markdown-filename-kebab-case/README.md), which covers both flavors. [The check README](../../README.md) points there.

## Read the failure

- `docs/api-design.md: filename must be camelCase`: rename the file to `docs/apiDesign.md` with `git mv`.
- `docs/Api_Design.md: filename must be camelCase`: rename it to `docs/apiDesign.md`.

Then update every link to the old name. Run the [markdown-links](../../../fitness-check-markdown-links/skills/markdown-links/SKILL.md) check to find the ones you missed. Only the repo's majority convention judges, so a repo of hyphenated names uses the [kebab-case flavor](../../../fitness-check-markdown-filename-kebab-case/skills/markdown-filename-kebab-case/SKILL.md).

## Check it first

Run `fitness markdown-filename-camel-case` from the repo root. Exit 0 means every name conforms.

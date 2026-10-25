---
name: markdown-filename-kebab-case
description: Fix a failing markdown-filename-kebab-case check on a markdown file name. Use when a doc is refused because its name is not kebab-case.
---

# Name markdown files in kebab-case

A markdown basename uses lowercase words joined by single hyphens, such as `api-design.md`. [The check README](../../go/cmd/fitness-check-markdown-filename-kebab-case/README.md) lists the exempt root docs and the names GitHub requires.

## Read the failure

- `docs/Api_Design.md: filename must be kebab-case`: rename the file to `docs/api-design.md` with `git mv`.
- `docs/releaseNotes.md: filename must be kebab-case`: rename it to `docs/release-notes.md`.

Then update every link to the old name. Run the [markdown-links](../markdown-links/SKILL.md) check to find the ones you missed. Only the repo's majority convention judges, so a repo of camelCase names uses the [camelCase flavor](../markdown-filename-camel-case/SKILL.md).

## Check it first

Run `fitness markdown-filename-kebab-case` from the repo root. Exit 0 means every name conforms.

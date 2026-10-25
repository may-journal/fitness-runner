---
name: changelog
description: Fix a failing changelog check on a changelog entry file. Use when an entry file is refused for its heading or its name.
---

# Fix a changelog entry file

Each file in `changelog/` holds one `### yyyy.mm.dd.HHMM` heading with its bullets, and its name starts with that heading's date. [The check README](../../README.md) states the rules and the [entry files](../../README.md#entry-files) section shows the shape.

## Read the failure

- `changelog/2026-05-01-0000-readme.md must hold one ### yyyy.mm.dd.HHMM heading and its bullets, nothing else`: keep one dated heading and its bullets. Remove any other heading or prose.
- `changelog/readme.md must be named changelog/2026-05-01-0000-readme.md: its heading's date, then the change`: rename the file so its date matches the heading, with dots read as hyphens, then a kebab-case change.

A file in a release tag is final, so add a new entry file for later changes.

## Check it first

Run `fitness changelog` from the repo root. Exit 0 means every entry file passes.

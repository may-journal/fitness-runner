---
name: strip-front-matter
description: Delete the old fitness keys (fitnessFunctions, relatedConfigurations) from a repo's Markdown files, keeping every body and any front matter other tools need. Use when a repo still carries them, or before adding a Markdown file.
---

# Strip fitness front matter

A Markdown file needs no front matter, as [docs/checks.md](../../docs/checks.md) says. No check reads `fitnessFunctions` or `relatedConfigurations`, so never add either to a new file, and drop any that remain.

## Rule

Run `fitness strip-front-matter` from the repo root, in your own work tree, then commit the result with a changelog entry. Do not edit the blocks by hand.

- It reads every tracked `.md` file: docs, changelog entries, templates and READMEs.
- It deletes both keys from each block, plain or inside the PR template's HTML comment.
- A block left with no key goes, and the body below it stays byte for byte.
- Other keys stay, such as an issue template's `labels` or a skill's `description`.
- It prints `stripped <file>` per changed file, naming the keys a block keeps, then a count.
- Review `git diff` before you commit.

## Why

A tool, not a hand edit, keeps every body untouched and every other tool's keys in place. [Requirement 0096](../../docs/requirements/0096-engine-strip-front-matter.md) states what it promises, and its tests prove it on a real repo.

## Example

```console
$ fitness strip-front-matter
stripped docs/guide.md
stripped changelog/2026-10-11-0900-one-change.md
stripped .github/ISSUE_TEMPLATE/bug.md: front matter keeps name, about, title, labels, assignees
fitness strip-front-matter: stripped 3 files
```

Then run `fitness` once. The [no-fitness-front-matter](../../go/cmd/fitness-check-no-fitness-front-matter/README.md) check fails any file that still holds either key, so a leftover cannot land. Files under `testdata` or `tests/examples` stay out of its scope as fixtures. The [changelog](../../go/cmd/fitness-check-changelog/README.md), [doc-template](../../go/cmd/fitness-check-doc-template/README.md) and [release-changelog](../../go/cmd/fitness-check-release-changelog/README.md) checks pass with or without front matter.

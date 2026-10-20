---
name: strip-front-matter
description: Remove the fitness front matter (fitnessFunctions, relatedConfigurations) from a repo's Markdown files, keeping every body and any front matter other tools need. Use when a repo adopts optional front matter or asks to drop the old fitness keys.
---

# Strip fitness front matter

Front matter is optional, as [docs/checks.md](../../docs/checks.md) says. No check reads `fitnessFunctions` or `relatedConfigurations` any more, so a repo can drop those blocks.

## Rule

Run `fitness strip-front-matter` from the repo root, in your own work tree, then commit the result with a changelog entry. Do not edit the blocks by hand.

- It reads every tracked `.md` file: docs, changelog entries, templates and READMEs.
- A block holding only fitness keys goes, plain or inside the PR template's HTML comment. The body below it stays byte for byte.
- A block that also holds another key stays whole, and the output names that key.
- Issue templates and `SKILL.md` files keep theirs, since GitHub and agents read them.
- It prints `stripped <file>` or `kept <file>: …` per file, then a count. Review `git diff` before you commit.

## Why

A tool, not a hand edit, keeps every body untouched and every other tool's keys in place. [Requirement 0096](../../docs/requirements/0096-engine-strip-front-matter.md) states what it promises, and its tests prove it on a real repo.

## Example

```console
$ fitness strip-front-matter
stripped docs/guide.md
stripped changelog/2026-10-11-0900-one-change.md
kept .github/ISSUE_TEMPLATE/bug.md: front matter also holds name, about, title, labels, assignees
fitness strip-front-matter: stripped 2 files, kept 1
```

Then run `fitness` once. The [changelog](../../go/cmd/fitness-check-changelog/README.md), [doc-template](../../go/cmd/fitness-check-doc-template/README.md) and [release-changelog](../../go/cmd/fitness-check-release-changelog/README.md) checks pass with or without front matter.

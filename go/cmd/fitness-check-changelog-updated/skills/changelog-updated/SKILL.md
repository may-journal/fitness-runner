---
name: changelog-updated
description: Fix a failing changelog-updated check on a commit. Use when a commit is refused for a missing, second, or unrelated changelog entry.
---

# Add the changelog entry to the commit

A branch keeps one entry file in `changelog/`. Its added lines share at least three words with the rest of the staged diff. [The check README](../../README.md) states every case, and the [changelog skill](../../../fitness-check-changelog/skills/changelog/SKILL.md) covers the file format.

## Read the failure

- `Stage a changelog/<yyyy-mm-dd-HHMM>-<change>.md entry file that mentions your staged changes`: create the entry file and stage it with your change.
- `this branch already has its entry changelog/2026-05-01-0000-readme.md; add your bullets to it instead of a new entry: changelog/2026-05-02-0000-more.md`: move your bullets into the first entry file and unstage the new one.
- `Changelog entry file additions should mention at least 3 words from your staged changes (found 1: sync)`: reword the bullets to name what the diff changes. The next line, `e.g. use words like: ...`, lists words from the diff.
- `<path> shipped in <tag>, so its notes are published; leave it as it is and add a new changelog/<yyyy-mm-dd-HHMM>-<change>.md entry file instead`: restore the released file and put your bullets in a new entry file.

## Check it first

Stage your files, then run `fitness changelog-updated` from the repo root. Exit 0 means it passes.

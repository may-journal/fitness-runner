---
name: cspell
description: Fix a failing cspell check on a tracked file. Use when a word is reported unknown or forbidden.
---

# Fix a misspelled word

Every tracked file is spell-checked, except generated files and lockfiles. [The check README](../../README.md) describes the dictionaries and the config.

## Read the failure

- `docs/a.md:12:5 - Unknown word (<word>)`: correct the spelling, or reword to a known word.
- `docs/a.md:12:5 - Forbidden word (entry)`: the word matches a `flagWords` entry in `cspell.json`. Use another word.
- `cspell.json sets ignorePaths; cspell checks every tracked file, so remove ignorePaths and fix the findings instead`: delete `ignorePaths` and fix each word it hid.

Fix the file. Do not add the word to an ignore list or skip the path. See [fix the work, not the limit](../../../../../docs/architecture/adr/0001-fix-the-work-not-the-limit.md).

## Check it first

Run `fitness cspell` from the repo root. Exit 0 means no unknown or forbidden words.

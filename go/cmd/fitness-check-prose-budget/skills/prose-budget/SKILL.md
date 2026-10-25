---
name: prose-budget
description: Fix a failing prose-budget check on a markdown file. Use when a doc is refused for too many words, sentences, paragraphs or list items.
---

# Cut markdown prose to its budget

Each markdown section gets a fixed amount of flowing prose. [The check README](../../README.md) lists the six limits. Headings, tables and fenced code do not count.

## Read the failure

- `docs/a.md: a sentence has 31 words (max 23): "…"`: split it or cut it.
- `docs/a.md: a paragraph has 5 sentences (max 4): "…"`: trim the paragraph, or start a new one.
- `docs/a.md: section "Setup" has 340 prose words (max 300)`: cut filler, or move detail into a table or a linked doc.
- `docs/a.md: section "Setup" has 4 paragraphs (max 3)`: merge or drop paragraphs.
- `docs/a.md: a list item has 25 words (max 23): "…"`: shorten the item.
- `docs/a.md: a list has more than 8 items at level 1: "…"`: group items into a table, or drop the weakest.

Fix the writing. Do not raise a limit in `.fitnessrc.json`.

## Check it first

Run `fitness prose-budget` from the repo root. Exit 0 means every file is within budget.

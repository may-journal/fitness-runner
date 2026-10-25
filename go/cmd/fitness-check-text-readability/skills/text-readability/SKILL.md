---
name: text-readability
description: Fix a failing text-readability check on a markdown file. Use when a doc gets a readability alarm.
---

# Make dense prose readable

A file fails when two of three readability formulas sit in alarm territory. [The check README](../../README.md) explains the masking and the bands. The [research note](../../../../../docs/research/0001-prose-cognitive-complexity.md) explains why.

## Read the failure

- `docs/a.md: readability alarm on 412 prose words — …`: two of the three scores are over their alarm. The file has long sentences or long words.

The check adds a `to fix:` line. Follow it:

- Split sentences over about 25 words. Semicolon and dash chains count as one sentence, so end them with periods.
- Wrap identifiers, file names and check names in backticks. Code spans are masked before scoring.
- Move quoted tool output and name lists into fenced code blocks.

Fix the prose. Do not change the bands in `.fitnessrc.json`.

## Check it first

Run `fitness text-readability` from the repo root. Add `--report` to print every file's scores. Exit 0 means no file is in alarm.

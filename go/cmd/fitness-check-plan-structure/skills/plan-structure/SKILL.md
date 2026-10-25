---
name: plan-structure
description: Fix a failing plan-structure check on a Plan issue. Use when the Plan issue check comments a failure, or before filing a Plan.
---

# Fix a Plan issue

A Plan is an issue under the `Plan` label. It has a one-line blockquote pitch, a `## Background`, and a `## What needs to happen` checklist, and nothing else. [The check README](../../README.md) states every rule and the [Plan template](../../../../../.github/ISSUE_TEMPLATE/plan.md) shows the shape. Start a Plan from that template.

## Read the failure

Edit the issue body, then wait for the bot to recheck.

- `Missing pitch: add a one-line blockquote (> …) before the first ## section`: put `> value in one sentence` first.
- `Pitch is still the template placeholder; write the real one-line value`: replace `REPLACE-ME`.
- ``Missing `## Background` section``: add the heading and one to three short paragraphs.
- ``Unexpected `## Notes` section — a plan has only Background and What needs to happen``: fold the text into Background or delete it.
- ``Remove the `## Open questions` section — turn unknowns into checklist steps``: write each unknown as a step.
- ``What needs to happen` has no checklist item (- [ ] …)``: add `- [ ]` items. A checkbox inside a code fence never counts.

## Check it first

Run `fitness plan-structure --body-file plan.md` on the body saved to a file in your own work tree, or pipe it on stdin with `--body-file -`. Exit 0 means the bot will pass it.

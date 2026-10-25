---
name: pr-structure
description: Fix a failing pr-structure check on a pull request description. Use when pr-check comments a failure, or before opening a PR.
---

# Fix a pull request description

A PR description has a one-line blockquote summary, a `## Background`, and a `## Changelog` with one bullet linking the PR's entry file. [The check README](../../README.md) states every rule and the [PR template](../../../../../.github/PULL_REQUEST_TEMPLATE.md) shows the shape. The diff says what changed, so the description carries only why.

## Read the failure

Edit the PR description, then wait for the bot to recheck.

- `Missing pitch: add a one-line blockquote (> …) before the first ## section`: start with `> the value in one sentence`.
- `Pitch is still the template placeholder; write the real one-line value`: replace `REPLACE-ME`.
- ``Missing `## Background` section``: add it with the why.
- ``Unexpected `## Testing` section — a PR has only Background and Changelog``: delete the section. Testing belongs in a PR comment.
- ``Changelog` has no list item (- …)``: add a bullet linking the entry file on your branch. Never copy its bullets.

## Check it first

Run `fitness pr-structure --body-file pr.md` on the description saved in your own work tree. The `pr-check` workflow also checks the changelog link against the diff.

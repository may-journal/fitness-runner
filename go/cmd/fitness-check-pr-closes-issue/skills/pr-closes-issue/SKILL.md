---
name: pr-closes-issue
description: Fix a failing pr-closes-issue check, where a PR description lacks a closing keyword. Use when pr-check says the PR closes nothing, or before opening a PR.
---

# Make a pull request close its issue

Every PR closes at least one issue on merge, and a Plan's own closing targets too. [The check README](../../README.md) lists GitHub's closing keywords and the three rules.

## Read the failure

Edit the PR description, then wait for the bot to recheck.

- `PR description has no GitHub closing keyword (close/fix/resolve + #NN)`: write `Closes #NN`. `addresses`, `part of` and a bare `#NN` never close. A keyword inside code never counts.
- `PR says it implements #5 but no closing keyword (close/fix/resolve + #5) closes it`: close `#5` too, or stop saying the PR implements it.
- `PR closes a Plan that closes #40, so the PR must also close #40`: add `Closes #40`.
- `unchecked item` on a closed issue: tick the finished boxes in that issue, or stop closing it.

## Check it first

Run `fitness pr-closes-issue --body-file pr.md` on the description saved in your own work tree. Add `--require-close=12,40` to mimic a Plan 12 that closes 40.

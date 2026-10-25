---
name: issue-checklist
description: Fix an issue-checklist failure, where an issue a PR closes still has unchecked boxes. Use when pr-check or close-check names an unchecked item.
---

# Tick the finished boxes

Closing an issue with unchecked items would close unfinished work. [The check README](../../README.md) states the rule: one error per unchecked `- [ ]` item, with fenced code skipped.

## Read the failure

- `unchecked item: "Write the tests."`: the issue still has `- [ ] Write the tests.`

Pick one fix, then rerun the bot.

- The work is done: edit the issue and tick the box to `- [x]`.
- The work is not done: finish it, or remove the closing keyword so the PR no longer closes the issue.
- The step is dropped: delete the line from the issue body.

After a merge, `fitness close-check` reopens an issue that closed with unchecked items, so tick before merging.

## Check it first

Run `fitness issue-checklist --body-file issue.md` on the body saved in your own work tree.

---
name: epic-structure
description: Fix a failing epic-structure check on an Epic issue. Use when the Epic issue check comments a failure, or before filing an Epic.
---

# Fix an Epic issue

An Epic is an issue under the `Epic` label. It has a pitch, a `## Background`, and a `## What needs to happen` list of plain issue references in run order. [The check README](../../README.md) states every rule and the [Epic template](../../../../../.github/ISSUE_TEMPLATE/epic.md) shows the shape.

## Read the failure

Edit the issue body, then wait for the bot to recheck.

- `item 1 uses a checkbox; list the issue as a plain reference`: write `#22`, not `- [ ] #22`.
- `item 2 has an emoji; GitHub shows each issue's state, so drop it`: delete the emoji and any status word.
- `item 3 (#24) is listed before #25, which it runs after`: move `#25` above `#24`.
- `tables are not allowed`: replace the table with the list.
- A list over the item cap fails prose-budget, not this check: split it under `###` headings.
- `N is listed but is not a sub-issue of this epic`: link the issue as a sub-issue, or drop it from the list. Every sub-issue must be listed too.

## Check it first

Run `fitness epic-structure --body-file epic.md` on the body saved in your own work tree. That judges the body only, since checks never call GitHub. The bot also compares the list with the sub-issues.

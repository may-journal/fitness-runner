---
name: comment-structure
description: Fix a failing comment-structure check on an issue, PR or review comment. Use when comment-check refuses a comment.
---

# Shorten and clean a comment

A comment stays short and free of agent noise. [The check README](../../README.md) lists every rule and the budgets.

## Read the failure

Edit the comment text in place, then wait for the bot to recheck.

- `(comment): 140 words outside tables, code, and diagrams; the budget is 100, so cut it or move detail into a table`: cut words.
- `(comment): 2 tables besides callout tables; the budget is 1`: keep one table.
- `(comment): line 4: mermaid diagram has no numbered callouts; number its nodes, as in A[1 Step], and add a # | Element | Description | Why table`: number the nodes and add the table.
- `(comment): line 2: "Update:": change-log note; edit the text in place and drop the note`: rewrite the text and drop the note.
- `(comment): line 3: "<system-reminder": agent frame; post only the finished text`: remove the frame.
- `(comment): line 5: "claude-sonnet-5": model identifier; name the agent, not the model`: name the agent.
- `carries a fitness marker; only fitness writes those, so remove it`: delete the marker.

## Check it first

Run `fitness comment-check --body-file comment.md` on the draft saved in your own work tree. Exit 0 means it passes.

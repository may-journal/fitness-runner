---
name: markdown-links
description: Fix a failing markdown-links check on a markdown file. Use when a relative link is reported broken.
---

# Repair a broken relative link

Every relative link in a markdown file must resolve to a real file or directory. [The check README](../../README.md) lists what is checked. Absolute URLs and `#anchor` links are never judged.

## Read the failure

- `docs/competition.md:11: broken relative link: ./README.md`: the target does not exist from that file's folder.

Open the file at the reported line, then pick the cause:

- The target moved or was renamed: point the link at its new path.
- The path is wrong for the folder: count the `../` steps from the linking file.
- The target was deleted on purpose: remove the link, or link what replaced it.

## Check it first

Run `fitness markdown-links` from the repo root. Exit 0 means every relative link resolves.

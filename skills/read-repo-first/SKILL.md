---
name: read-repo-first
description: Act on the read-repo-first box fitness prints before a run. Use when a boxed reminder about the repo's decisions and enabled checks appears.
---

# Read the repo first

`read-repo-first` never fails. It prints a box asking whether you read the decisions logged in the repo and the fitness checks enabled for it. [The check README](../../go/cmd/fitness-check-read-repo-first/README.md) states what the box shows.

## What to do

- Read the repo's `AGENTS.md` and `docs/` before changing code.
- Open the README listed beside each check in the box's table. It states the rule, so fix the code to satisfy it.
- Use the skill named for a failing check when one exists.
- Never pass `--no-verify` to git, as the box's note says.

Exit 0 with the box printed is the pass. Nothing needs fixing.

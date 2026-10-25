---
name: local-branch
description: Fix a failing local-branch check, which judges a branch against its open PR and Plan. Use when fitness or a push fails with a pr-check or plan-check line for the branch.
---

# Fix the branch metadata

`local-branch` runs the PR and Plan checks on this branch's open PR and its linked issues, without posting anything. [The check README](../../README.md) states what it reads and when it skips.

## Read the failure

- `pr-check #1` followed by a PR title or body line: fix that PR with [semantic-commit](../../../fitness-check-semantic-commit/skills/semantic-commit/SKILL.md), [pr-structure](../../../fitness-check-pr-structure/skills/pr-structure/SKILL.md) or [pr-closes-issue](../../../fitness-check-pr-closes-issue/skills/pr-closes-issue/SKILL.md), then rerun.
- `plan-check #12` followed by a Plan line: fix the issue with [plan-structure](../../../fitness-check-plan-structure/skills/plan-structure/SKILL.md) or [issue-checklist](../../../fitness-check-issue-checklist/skills/issue-checklist/SKILL.md).
- `no closing issue links`: the PR body closes nothing. Add `Closes #NN`.
- `multiple open PRs`: close the extra PR for this branch.
- `cannot validate linked issue #11` or `HTTP 401`: GitHub could not answer. Run `gh auth status`, then `fitness branch-check` again.
- `no open PR` and `detached HEAD` skip with exit 0. Open the PR to get judged.

## Check it first

Run `fitness local-branch` (alias `branch-check`, the name its failures print) on your branch. It reads GitHub live and changes nothing.

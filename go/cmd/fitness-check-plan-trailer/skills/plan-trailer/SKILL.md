---
name: plan-trailer
description: Fix a failing plan-trailer check on a commit message. Use when a commit is refused for a misspelled Plan reference, or when linking a commit to its Plan.
---

# Spell the Plan trailer

A commit may name its Plan with a line reading exactly `Plan #<number>`. The trailer is optional, but a spelling that is present must be exact. [The check README](../../README.md) states the rule.

## Read the failure

- `plan trailer "Plan 63" must read exactly "Plan #<number>"`: the message also fails for spellings such as `plan #63` or `Plan: 63`. Rewrite each as `Plan #63`.
- Prose such as "Plan the rollout in 3 steps" is never judged, so leave it.

Put the trailer on the branch's first commit only. [issue-link-once](../../../fitness-check-issue-link-once/skills/issue-link-once/SKILL.md) fails a second commit that links the same Plan.

## Check it first

Run `fitness plan-trailer --message "$(cat msg.txt)"` on the proposed message. With no `--message` it judges the HEAD commit. Fix a HEAD you have not pushed with `git commit --amend`.

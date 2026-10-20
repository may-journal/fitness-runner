---
name: issue-link-once
description: Fix a failing issue-link-once check, where a second commit or the PR body links an issue already linked. Use when a commit is refused for a repeated issue link, or CI fails the commits.
---

# Link an issue from one commit only

Each commit that links an issue adds an entry to its timeline, so a branch links each issue once. [The check README](../../go/cmd/fitness-check-issue-link-once/README.md) states what counts as a link and where it runs.

## Read the failure

- `may-journal/example#12 is already linked by commit`, then the sha and subject of the earlier commit: remove the link from the new message. The first commit keeps `Plan #12`.
- `acme/app#12 is already linked by open PR #7` or `#12 is already linked by open PR #9`: the PR body already closes it. Take the link out of the commit message.
- `commit abc1234 ("fix(app): more"): #12 is already linked by commit` in CI: a pushed commit repeats the link. Rewrite it and push with `--force-with-lease`.

Rewrite only your own branch. A branch holding an owner approval is not yours to rewrite: ask the owner first.

## Check it first

Run `fitness issue-link-once --message "$(cat msg.txt)"` on the proposed message in your branch. A different issue, or the same number in another repo, passes.

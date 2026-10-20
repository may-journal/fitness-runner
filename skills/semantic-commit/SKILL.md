---
name: semantic-commit
description: Fix a failing semantic-commit check on a commit subject or PR title. Use when a commit or PR title is refused for its format.
---

# Write a conventional commit subject

A subject reads `type(scope): description`, and the scope is required. [The check README](../../go/cmd/fitness-check-semantic-commit/README.md) lists what it judges. The PR title gets the same check as a commit subject.

## Read the failure

- `Commit message: "feat: add endpoint" — use type(scope): description (types: feat, fix, docs, style, refactor, perf, test, build, ci, chore, revert)`: the scope is missing. Write `feat(api): add endpoint`.
- The same line for `Add notes`: pick a type and scope, as in `docs(notes): add notes`.
- `No commit message to validate; use type(scope): description`: the message was empty.

Fix a HEAD you have not pushed with `git commit --amend`. Fix a PR title by editing the title, then wait for the bot. Merge commits are accepted as they are.

## Check it first

Run `fitness semantic-commit --message "feat(api): add endpoint"` with your subject. Exit 0 means it passes.

---
name: commit-attribution
description: Fix a failing commit-attribution check, where a commit lacks the AI-Tools and AI-Models trailers. Use when a commit is refused for a missing trailer.
---

# Disclose the AI tooling in a commit

A repo that uses the convention requires two trailers on every commit: `AI-Tools:` and `AI-Models:`, each with a value on its own line. [The check README](../../README.md) states the rule and what exempts merge and revert commits.

## Read the failure

- `commit message missing "AI-Models:" trailer` or `"AI-Tools:" trailer`: add that line below the body, such as `AI-Tools: Claude Code`.
- A key with no value, or one inside a sentence, does not count. Start its own line with it.
- `No commit message to validate; add "AI-Tools:" and "AI-Models:" trailers`: the message was empty. Write one.

Take the tool and model names from the attribution your session was given. Do not invent them.

## Check it first

Run `fitness commit-attribution --message "$(cat msg.txt)"` on the proposed message. Without `--message` it judges the HEAD commit, so fix a commit you have not pushed with `git commit --amend`.

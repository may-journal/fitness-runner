---
relatedConfigurations: ['../../../.fitnessrc.json']
---

# issue-link-once

Keeps Issue timelines readable ([#153](https://github.com/may-journal/fitness-runner/issues/153)). Every commit that links an Issue adds an entry to its timeline. So a branch links each Issue from one commit at most.

The `commit-msg` hook runs it on the proposed message, beside `semantic-commit` and `plan-trailer`. Without a context-inline `--message` it passes inert, so full-suite runs leave it alone.

## Behavior

- A link is `#NN`, `owner/repo#NN`, or an Issue or PR URL on github.com. A bare `#NN` belongs to origin's repo.
- Fail: an earlier commit on the branch, since its merge base with origin's default branch, links the same Issue.
- Fail: the branch's open PR body links the Issue, since the PR already shows on its timeline.
- Pass: links to Issues nothing else on the branch links yet, and messages with no links.
- Comment lines and anything below the scissors line are ignored, as git drops them.
- An amend leaves out HEAD, the commit it replaces.
- With no open PR, or no network, it warns and skips the PR rule.

Every PR must close an Issue, so in practice only commits pushed before the PR opens link one. The squash commit on `main` carries the PR body, and that link stays.

## Contributing

This README is the canonical description for this check, a self-contained `fitness-check-issue-link-once` binary. To change what counts as a link, extend the check and its tests here and keep this README in sync.

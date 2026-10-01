---
relatedConfigurations: ['../../../.fitnessrc.json']
---

# issue-link-once

Keeps Issue timelines readable ([#153](https://github.com/may-journal/fitness-runner/issues/153)). Every commit that links an Issue adds an entry to its timeline. So a branch links each Issue from one commit at most.

## Behavior

- A link is `#NN`, `owner/repo#NN`, or an Issue or PR URL on github.com. A bare `#NN` belongs to origin's repo.
- Fail: an earlier commit on the branch, since the base branch, links the same Issue.
- Fail: the open PR body links the Issue, and the commit is newer than the PR.
- Comment lines and anything below the scissors line are ignored, as git drops them.

## Where it runs

The `commit-msg` hook judges the proposed message. An amend leaves out HEAD, the commit it replaces. With no open PR, or no network, it skips the PR rule.

On a pull request, the org `fitness-suite` workflow judges every branch commit, oldest first, in every repo ([#157](https://github.com/may-journal/fitness-runner/issues/157)). The one link written before the PR opened passes, judged by author date, which a rebase keeps. A failure stays red until you rewrite that commit without the link and force-push with `--force-with-lease`.

Anywhere else, such as a local suite run or a push to `main`, it passes inert.

## Contributing

This README is the canonical description for this check, a self-contained `fitness-check-issue-link-once` binary. To change what counts as a link, extend the check and its tests here and keep this README in sync.

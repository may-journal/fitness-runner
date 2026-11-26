### 2026.11.23.1439

> A PR stacked on an open PR is judged on the changelog entry it adds itself.

- Fix: `pr-check` no longer counts an entry file that a lower open PR in the stack adds.
- Fix: a lower PR is an open PR whose head commit is among the PR's commits, never a line in its description.
- Fix: `branch-check` judges the local branch the same way before the push.
- Fix: a stacked PR that adds no entry of its own no longer has to link the lower PR's entry.

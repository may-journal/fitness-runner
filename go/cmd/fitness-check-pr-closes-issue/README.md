---
relatedConfigurations: ['../../../.github/workflows/pr-check.yml']
---

# pr-closes-issue

Every PR must close at least one issue on merge ([#69](https://github.com/may-journal/fitness-runner/issues/69)). GitHub auto-closes a linked issue only for a fixed set of keywords, so a PR that merely references an issue leaves it open — this check fails that.

Runs in [pr-check.yml](../../../.github/workflows/pr-check.yml) beside `pr-structure`. It reads the PR body from stdin, a `--body-file` path, or the context-inline `--body`, and passes inert with no input.

## Behavior

Two rules run over the PR description:

- Every PR carries at least one GitHub closing keyword (`close`/`fix`/`resolve` and their tenses) followed by `#NN`. A body with only references, or none, fails.
- Every issue the PR says it implements (`Implements #NN` or a `Plan #NN` line) must be among the closed issues; an unclosed one fails, named individually.

The full closing-keyword set is `close`, `closes`, `closed`, `fix`, `fixes`, `fixed`, `resolve`, `resolves`, `resolved`. References like `addresses #12`, `part of #12`, or a bare `#12` never count as a closure.

There is no chore or docs exemption — every PR is held to the first rule.

## Contributing

This README is the canonical description for this check. The closing-keyword list is GitHub's fixed set; keep it in sync here and in `closingKeywords` if GitHub's set ever changes.

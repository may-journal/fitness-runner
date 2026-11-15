### 2026.11.15.1800

> The pre-push gate judges the PR body again in repos with no installed checks.

- Fix: the pre-push hook `fitness init` writes sets `FITNESS_WITH_CHECKS=pinned`, as pre-commit and commit-msg do.
- Fix: checks the engine fetches reach the checks it starts, so local-branch finds the PR body checks.
- Test: requirement 0170 pushes from a pinned repo with no installed checks, through both hooks.

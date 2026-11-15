### 2026.11.15.1800

> The pre-push gate and the GitHub commands judge PR bodies again with no checks installed.

- Fix: the pre-push hook `fitness init` writes sets `FITNESS_WITH_CHECKS=pinned`, and fetched checks reach local-branch.
- Fix: `pr-check`, `issue-check`, `plan-check`, `comment-check` and the other GitHub commands fetch missing pinned checks.
- Fix: no PR body check reports "no result from the check" for lack of an installed check.
- Test: requirement 0170 pushes from a pinned repo with no installed checks, through both hooks.
- Test: requirement 0171 runs `fitness pr-check` from a pinned repo with none installed.

# 0170 Hooks pre-push checks

## Why

The engine holds no checks, and the push gate starts local-branch as its own process. Pre-push must hand that process the checks it installs or fetches. Otherwise each PR body check gives no result, and the push is refused.

## Measurement

Pushes from a pinned repo with no checks installed that judge the PR body
-
Pushes from a pinned repo with no checks installed

## Requirements

- 0170.1
    - Given Developer has a repo pinning every check, none installed, and a PR with bold text
        - When I push with the hook `fitness init` writes, or with one that sets no `FITNESS_WITH_CHECKS`
            - Then the push is refused naming the bold text, and every PR body check gives a result

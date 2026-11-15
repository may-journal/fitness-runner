# 0171 GitHub commands fetch checks

## Why

The engine holds no checks, and a repo's own tests run `fitness pr-check` with none handed down. So each GitHub command fetches the pinned checks it judges a body with, or every body check gives no result.

## Measurement

GitHub command runs with no checks installed that judge the body
-
GitHub command runs with no checks installed

## Requirements

- 0171.1
    - Given Developer has a repo pinning every check, none installed, and a PR whose title is not a semantic commit
        - When I run `fitness pr-check` on its event
            - Then it fetches the checks and fails naming the title's form, and every PR body check gives a result

# 0014 Pull Requests

## Why

A pull request merges only when its title, description, and the issues it closes are all ready.

## Measurement

pull requests merged with a clean pr-check
-
pull requests merged

## Requirements

- 0014.1
    - Given Developer has a draft PR that follows the template, linking its changelog entry, and closes a finished Plan
        - When Actions and branch-check judge it
            - Then both report green, and local validation links targets without writing GitHub state
- 0014.2
    - Given Developer has a PR whose title is not a semantic commit
        - When pr-check judges it
            - Then it reports red quoting the title
- 0014.3
    - Given Developer has a PR that closes a Plan with an unchecked item
        - When Actions and branch-check judge it
            - Then it reports red naming the item
- 0014.4
    - Given Developer has a PR that leaves open an issue its Plan closes
        - When pr-check judges it
            - Then it reports red naming that issue
- 0014.5
    - Given Developer has a manual workflow run
        - When pr-check runs
            - Then it judges every open PR
- 0014.6
    - Given Developer has a PR closing an inaccessible issue
        - When I check it locally and through Actions
            - Then both fail naming that issue
- 0014.7
    - Given Developer has fixed a PR's title and body after its check failed
        - When Actions re-runs pr-check on the saved event
            - Then it judges the title and body GitHub holds now, and passes
- 0014.8
    - Given Developer has a PR check whose runner cannot reach GitHub
        - When pr-check runs
            - Then it warns and judges the title and body the event carries

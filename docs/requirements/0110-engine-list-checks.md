# 0110 List Checks

## Why

One command tells me every check, what it enforces, what it reads and whether it runs here. I need no checkout of fitness.

## Measurement

checks that `fitness list` names with a summary and a scope
-
checks in the catalog

## Requirements

- 0110.1
    - Given AI agent has a repo with fitness installed
        - When it runs `fitness list`
            - Then every built-in check shows once with its summary, scope and budget
- 0110.2
    - Given AI agent has a repo whose config disables a check under external policy
        - When it runs `fitness list --json`
            - Then that check reads as not enabled and every other check as enabled
- 0110.3
    - Given AI agent has a repo with a local check
        - When it runs `fitness list`
            - Then the local check shows with the summary its describe handshake gives
- 0110.4
    - Given AI agent has a repo with a local check whose describe handshake fails
        - When it runs `fitness list`
            - Then that check shows with a note saying why, and every other check still shows
- 0110.5
    - Given AI agent has a repo whose own Go check build removes the runner's file
        - When it runs `fitness list`
            - Then every built-in check still shows its summary and scope, read without starting it

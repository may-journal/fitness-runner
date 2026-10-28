# 0095 Auto-merge stacks

## Why

Once every pull request in my native stack is approved and green, the whole stack merges, so nobody merges by hand. Until then, each pull request says which one waits and why. A refused or failed merge turns the check red with GitHub's answer.

## Measurement

Approved, green native stacks merged by Auto-merge
-
Approved, green native stacks

## Requirements

- 0095.1
    - Given Developer has a native stack whose pull requests are all approved, green and mergeable
        - When Auto-merge runs on one of them and GitHub merges the stack
            - Then it merges the top pull request at its head without bypassing rules, and comments on each that the stack merged
- 0095.2
    - Given Developer has a native stack with a pull request that is not approved
        - When Auto-merge runs on it
            - Then it passes, merges nothing, and comments on each pull request which one waits for approval
- 0095.3
    - Given Developer has a native stack with a pull request whose checks are red
        - When Auto-merge runs on it
            - Then it passes, merges nothing, and comments on each pull request which one is not green
- 0095.4
    - Given Developer has an approved, green native stack
        - When GitHub refuses to merge its top pull request
            - Then it fails naming GitHub's answer and comments the reason on the top pull request
- 0095.5
    - Given Developer has an approved, green native stack
        - When GitHub accepts the merge and its result then reads failed
            - Then it fails naming GitHub's answer and comments the reason on the top pull request
- 0095.6
    - Given Developer has a pull request
        - When someone approves it, approves a fork's pull request, or reviews without approving
            - Then Auto-merge runs on the first and skips the others before a runner starts

# 0010 Push Plan

## Why

Every feature I push traces to a Plan a person approved, so no unplanned work reaches review. My branch's open PR names that Plan by closing it. The Plan may live in another repo of my owner, such as an org-wide Plan in `.github`.

## Measurement

feature pushes whose open PR closes an approved Plan
-
feature pushes with an open PR

## Requirements

- 0010.1
    - Given Developer has the shared hooks, no fitness on the PATH, and a branch whose open PR closes an approved Plan
        - When I push it
            - Then the push goes through, with the repo's owner and org members allowed when planApprovers is unset
- 0010.4
    - Given Developer has a feature branch with no open PR yet
        - When I push it
            - Then the push goes through with one info line, since the PR check gates the Plan later
- 0010.6
    - Given Developer has pushed a branch
        - When I delete it from the remote
            - Then the deletion goes through
- 0010.10
    - Given Developer has only chore or docs commits, even with feature commits merged in from main
        - When I push them
            - Then the push goes through
- 0010.11
    - Given Developer has feature commits on a branch whose PR closes one approved Plan twice, or a network that never answers
        - When I push them
            - Then the hook asks GitHub about that Plan once, or refuses once 30 seconds pass with one line naming the host
- 0010.12
    - Given Developer has external policy with plan-trailer and local-branch in disabledChecks
        - When I push a feature branch whose open PR closes no Plan
            - Then the push goes through without reading the PR
- 0010.13
    - Given Developer has a branch whose PR closes an approved Plan of origin's owner in another repo, as `Closes May-Journal/.github#144` or its URL
        - When I push it
            - Then the push goes through, having read that Plan once from the repo it names
- 0010.14
    - Given Developer has a branch whose PR closes no Plan despite a `Plan #NN` trailer, or a non-Plan issue, foreign or unapproved Plan
        - When I push it
            - Then the push is refused naming that issue and who may approve, or that the PR body must close the Plan

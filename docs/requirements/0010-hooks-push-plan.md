---
relatedConfigurations: ['../../.fitnessrc.json']
---

# 0010 Push Plan

## Why

Every feature I push traces to a Plan a person approved, so no unplanned work reaches review.

## Measurement

feature pushes that name an approved Plan
-
feature pushes

Source: `go/cmd/fitness/hook.go:26`

## Requirements

- 0010.1
    - Given Developer has the shared hooks, no fitness on the PATH, and a feature commit whose trailer names an approved Plan
        - When I push it
            - Then the push goes through
- 0010.4
    - Given Developer has a branch whose first pushed commit names an approved Plan
        - When I push a later feature commit with no trailer
            - Then the push goes through
- 0010.6
    - Given Developer has pushed a branch
        - When I delete it from the remote
            - Then the deletion goes through
- 0010.9
    - Given Developer has a feature commit with no Plan trailer, or one naming a non-Plan issue or an unapproved Plan
        - When I push it
            - Then the push is refused naming the missing trailer or that issue
- 0010.10
    - Given Developer has only chore or docs commits, even with feature commits merged in from main
        - When I push them
            - Then the push goes through
- 0010.11
    - Given Developer has several feature commits naming one approved Plan
        - When I push them
            - Then the hook asks GitHub about that Plan once
- 0010.12
    - Given Developer has external policy with plan-trailer and local-branch in disabledChecks
        - When I push a feature commit that names no Plan
            - Then the push goes through

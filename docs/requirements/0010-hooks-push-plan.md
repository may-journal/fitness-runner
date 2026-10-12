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
    - Given Developer has a feature commit whose trailer names an approved Plan
        - When I push it
            - Then the push goes through
- 0010.2
    - Given Developer has a feature commit whose trailer names an issue that is not a Plan
        - When I push it
            - Then the push is refused naming that issue
- 0010.3
    - Given Developer has a feature commit whose trailer names a Plan nobody approved
        - When I push it
            - Then the push is refused naming that Plan
- 0010.4
    - Given Developer has a branch whose first pushed commit names an approved Plan
        - When I push a later feature commit with no trailer
            - Then the push goes through
- 0010.5
    - Given Developer has feature commits from main merged into the chore branch
        - When I push it
            - Then the push goes through
- 0010.6
    - Given Developer has pushed a branch
        - When I delete it from the remote
            - Then the deletion goes through

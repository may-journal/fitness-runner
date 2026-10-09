---
relatedConfigurations: ['../../.fitnessrc.json']
---

# 0010 Checks Push Plan

## Why

Every feature I push traces to a Plan a person approved, so no unplanned work reaches review.

## Measurement

feature pushes that name an approved Plan
-
feature pushes

Source: `go/cmd/fitness/hook.go:26`

## Requirements

- 0010.1
    - Given a feature commit whose trailer names an approved Plan
        - When I push it
            - Then the push goes through
- 0010.2
    - Given a feature commit whose trailer names an issue that is not a Plan
        - When I push it
            - Then the push is refused naming that issue
- 0010.3
    - Given a feature commit whose trailer names a Plan nobody approved
        - When I push it
            - Then the push is refused naming that Plan
- 0010.4
    - Given a branch whose first pushed commit names an approved Plan
        - When I push a later feature commit with no trailer
            - Then the push goes through
- 0010.5
    - Given feature commits from main merged into my chore branch
        - When I push it
            - Then the push goes through
- 0010.6
    - Given a branch I pushed
        - When I delete it from the remote
            - Then the deletion goes through

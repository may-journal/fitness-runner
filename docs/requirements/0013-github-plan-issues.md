---
relatedConfigurations: ['../../.fitnessrc.json']
---

# 0013 Plan Issues

## Why

Every Plan shows one current verdict and label, so I see at a glance whether it is ready to approve.

## Measurement

Plan edits whose verdict and label match their body
-
Plan edits

Source: `go/internal/aftereffect/verdict.go:15`

## Requirements

- 0013.1
    - Given Developer has a Plan that follows the template
        - When plan-check judges it
            - Then it comments that the Plan looks good and labels it fitness-valid
- 0013.2
    - Given Developer has a Plan that breaks the template
        - When plan-check judges it
            - Then it comments the errors, labels it fitness-invalid, and exits 1
- 0013.3
    - Given Developer has a Plan that plan-check already judged, or one where only another author pasted that verdict
        - When it judges the same body again
            - Then it adds no comment to the first, and posts a real verdict on the second
- 0013.4
    - Given Developer has a failed Plan that is then fixed
        - When plan-check judges it
            - Then it comments a pass and hides the failed verdict
- 0013.5
    - Given Developer has a passing Plan edited so it still passes
        - When plan-check judges it
            - Then it updates its passing verdict in place
- 0013.6
    - Given Developer has a manual workflow run
        - When plan-check runs
            - Then it judges every open Plan
- 0013.9
    - Given Developer has a malformed Plan in both the PR and trailer, or from a fork whose PR base exists only upstream
        - When I run fitness branch-check
            - Then it reports the Plan body violation and reads the Plan once
- 0013.10
    - Given Developer has a Plan moved to another repo before plan-check reaches it
        - When plan-check judges it
            - Then it notes the Plan is gone and exits 0

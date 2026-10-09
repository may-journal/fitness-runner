---
relatedConfigurations: ['../../.fitnessrc.json']
---

# 0013 Checks Plan Issues

## Why

Every Plan shows one current verdict and label, so I see at a glance whether it is ready to approve.

## Measurement

Plan edits whose verdict and label match their body
-
Plan edits

Source: `go/internal/aftereffect/verdict.go:14`

## Requirements

- 0013.1
    - Given a Plan that follows the template
        - When plan-check judges it
            - Then it comments that the Plan looks good and labels it fitness-valid
- 0013.2
    - Given a Plan that breaks the template
        - When plan-check judges it
            - Then it comments the errors, labels it fitness-invalid, and exits 1
- 0013.3
    - Given a Plan that plan-check already judged
        - When it judges the same body again
            - Then it adds no comment
- 0013.4
    - Given a failed Plan that is then fixed
        - When plan-check judges it
            - Then it comments a pass and hides the failed verdict
- 0013.5
    - Given a passing Plan edited so it still passes
        - When plan-check judges it
            - Then it updates its passing verdict in place
- 0013.6
    - Given a manual workflow run
        - When plan-check runs
            - Then it judges every open Plan

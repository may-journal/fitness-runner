---
relatedConfigurations: ['../../.fitnessrc.json']
---

# 0049 Checks Plan Trailer

## Why

I can link a commit to its Plan Issue, and a misspelled link is caught before it lands.

## Measurement

plan references that read exactly `Plan #<number>`
-
plan references in commit messages

Source: `go/cmd/fitness-check-plan-trailer/main.go:50`

## Requirements

- 0049.1
    - Given a commit message with no plan reference
        - When plan-trailer checks it
            - Then it passes, since the trailer is optional
- 0049.2
    - Given a commit message ending in `Plan #63`
        - When plan-trailer checks it
            - Then it passes
- 0049.3
    - Given plan references spelled `Plan 63`, `plan #63`, and `Plan: 63`
        - When plan-trailer checks the message
            - Then it fails once per line, quoting it with the `Plan #<number>` fix
- 0049.4
    - Given body prose that starts with Plan, like "Plan the rollout in 3 steps"
        - When plan-trailer checks the message
            - Then it passes, since prose is not a trailer
- 0049.5
    - Given no message passed and a HEAD commit with a misspelled plan reference
        - When plan-trailer runs
            - Then it fails quoting the HEAD commit's line

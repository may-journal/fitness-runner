---
relatedConfigurations: ['../../.fitnessrc.json']
---

# 0056 Checks Semantic Commit

## Why

I read history and release notes by commit type and scope, so every subject must say what kind of change it is.

## Measurement

subjects judged by the semantic rules
-
subjects checked

Source: `go/cmd/fitness-check-semantic-commit/main.go:86`

## Requirements

- 0056.2
    - Given Developer has a subject with a type but no scope
        - When semantic-commit checks the message
            - Then it fails listing the accepted types
- 0056.7
    - Given Developer has an empty `--message`, or no git repo to read HEAD from
        - When semantic-commit runs
            - Then it fails saying there is no message
- 0056.9
    - Given Developer has a `Merge` subject, or any listed type with a scope over any body
        - When semantic-commit checks the message
            - Then it passes, judging only the first line
- 0056.10
    - Given Developer has a type outside the list, such as `Revert`, or a breaking mark such as `feat(api)!:`
        - When semantic-commit checks the message
            - Then it fails quoting the subject
- 0056.11
    - Given Developer has the shared hooks on a fresh branch whose parent subject has no scope
        - When I commit with a valid subject
            - Then the commit succeeds
- 0056.12
    - Given Developer has the shared hooks
        - When I commit with a subject that has no scope
            - Then commit-msg refuses the commit quoting that subject
- 0056.13
    - Given Developer has a branch, or local main, with a bad commit above bad history origin already holds
        - When I run the full Fitness suite
            - Then semantic-commit fails quoting only the commit origin lacks
- 0056.14
    - Given Developer has no fetched origin branch to start from
        - When semantic-commit runs with no message
            - Then it judges the HEAD commit subject

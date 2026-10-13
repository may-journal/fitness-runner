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

Source: `go/cmd/fitness-check-semantic-commit/main.go:72`

## Requirements

- 0056.1
    - Given Developer has a subject with any listed type and a scope
        - When semantic-commit checks the message
            - Then it passes
- 0056.2
    - Given Developer has a subject with a type but no scope
        - When semantic-commit checks the message
            - Then it fails listing the accepted types
- 0056.3
    - Given Developer has a type outside the list, such as git's own `Revert` subject
        - When semantic-commit checks the message
            - Then it fails quoting the subject
- 0056.4
    - Given Developer has a subject with a breaking mark, such as `feat(api)!:`
        - When semantic-commit checks the message
            - Then it fails quoting the subject
- 0056.5
    - Given Developer has a `Merge` subject
        - When semantic-commit checks the message
            - Then it passes
- 0056.6
    - Given Developer has a valid subject over a body that breaks the format
        - When semantic-commit checks the message
            - Then it passes, judging only the first line
- 0056.7
    - Given Developer has an empty `--message`, or no git repo to read HEAD from
        - When semantic-commit runs
            - Then it fails saying there is no message
- 0056.8
    - Given Developer has no message argument
        - When semantic-commit runs
            - Then it judges the HEAD commit subject

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
    - Given a subject with any listed type and a scope
        - When semantic-commit checks the message
            - Then it passes
- 0056.2
    - Given a subject with a type but no scope
        - When semantic-commit checks the message
            - Then it fails listing the accepted types
- 0056.3
    - Given a type outside the list, such as git's own `Revert` subject
        - When semantic-commit checks the message
            - Then it fails quoting the subject
- 0056.4
    - Given a subject with a breaking mark, such as `feat(api)!:`
        - When semantic-commit checks the message
            - Then it fails quoting the subject
- 0056.5
    - Given a `Merge` subject
        - When semantic-commit checks the message
            - Then it passes
- 0056.6
    - Given a valid subject over a body that breaks the format
        - When semantic-commit checks the message
            - Then it passes, judging only the first line
- 0056.7
    - Given an empty `--message`, or no git repo to read HEAD from
        - When semantic-commit runs
            - Then it fails saying there is no message
- 0056.8
    - Given no message argument
        - When semantic-commit runs
            - Then it judges the HEAD commit subject

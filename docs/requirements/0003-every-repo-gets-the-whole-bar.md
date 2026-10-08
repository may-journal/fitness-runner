---
relatedConfigurations: ['../../.fitnessrc.json']
---

# 0003 Every Repo Gets the Whole Bar

## Why

A new repo is held to every rule from its first commit, with nothing to pick or configure.

## Measurement

checks that pass clean in a repo they do not apply to
-
checks in the catalog

Source: `go/cmd/fitness/contract_test.go:95`

## Requirements

- 0003.1
    - Given an empty repo
        - When each check runs in it
            - Then every check that does not apply passes with zero files
- 0003.2
    - Given a new check binary
        - When the runner lists its catalog
            - Then the new check is in it, so every repo runs it
- 0003.3
    - Given a repo that turns off one check by name
        - When the runner builds its list
            - Then only that check is left out

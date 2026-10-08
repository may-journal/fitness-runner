---
relatedConfigurations: ['../../.fitnessrc.json']
---

# 0005 Checks Go Requirements

## Why

Every test proves a stated requirement and every requirement has exactly one test, so nothing is untested or tested twice.

## Measurement

acceptances owned by exactly one test
-
acceptances

Source: `go/cmd/fitness-check-requirements/owners.go:100`

## Requirements

- 0005.1
    - Given an acceptance that no test owns
        - When fitness runs
            - Then it fails
- 0005.2
    - Given an acceptance that two tests own
        - When fitness runs
            - Then it fails
- 0005.3
    - Given a test named for a removed acceptance
        - When fitness runs
            - Then it fails
- 0005.4
    - Given a test that proves no requirement
        - When fitness judges its file
            - Then it fails
- 0005.5
    - Given a Go repo with no requirement docs
        - When fitness runs
            - Then it fails
- 0005.6
    - Given a repo without Go
        - When fitness runs
            - Then the check passes with nothing to judge
- 0005.7
    - Given a scoped run that changes no doc or test
        - When fitness runs
            - Then the check skips
- 0005.8
    - Given an ID that an earlier commit deleted
        - When a doc brings it back
            - Then it fails

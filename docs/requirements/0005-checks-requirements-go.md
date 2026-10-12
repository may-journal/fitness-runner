---
relatedConfigurations: ['../../.fitnessrc.json']
---

# 0005 Checks Requirements Go

## Why

Every test proves a stated requirement and every requirement has exactly one test, so nothing is untested or tested twice.

## Measurement

acceptances owned by exactly one test
-
acceptances

Source: `go/cmd/fitness-check-requirements/owners.go:100`

## Requirements

- 0005.1
    - Given Developer has an acceptance that no test owns
        - When fitness runs
            - Then it fails
- 0005.2
    - Given Developer has an acceptance that two tests own
        - When fitness runs
            - Then it fails
- 0005.3
    - Given Developer has a test named for a removed acceptance
        - When fitness runs
            - Then it fails
- 0005.4
    - Given Developer has a test that proves no requirement
        - When fitness runs
            - Then it fails
- 0005.5
    - Given Developer has a Go repo with no requirement docs
        - When fitness runs
            - Then it fails
- 0005.6
    - Given Developer has a repo with no Go or Swift code
        - When fitness runs
            - Then the check passes with nothing to judge
- 0005.7
    - Given Developer has an unchanged Go test without a requirement
        - When fitness runs on staged or pull request changes
            - Then it fails even for unrelated changes
- 0005.8
    - Given Developer has an ID that an earlier commit deleted
        - When a doc brings it back
            - Then it fails

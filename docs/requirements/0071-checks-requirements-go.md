---
relatedConfigurations: ['../../.fitnessrc.json']
---

# 0071 Checks Requirements Go

## Why

A Go repo's tests prove its requirements, so each acceptance has exactly one test and nothing is untested or tested twice.

## Measurement

Go acceptances owned by exactly one test
-
Go acceptances

Source: `go/cmd/fitness-check-requirements/owners.go:100`

## Requirements

- 0071.1
    - Given Developer has an acceptance that no Go test owns
        - When fitness runs
            - Then it fails
- 0071.2
    - Given Developer has an acceptance that two Go tests own
        - When fitness runs
            - Then it fails
- 0071.3
    - Given Developer has a Go test named for a removed acceptance
        - When fitness runs
            - Then it fails
- 0071.4
    - Given Developer has a Go test that proves no requirement
        - When fitness runs
            - Then it fails
- 0071.5
    - Given Developer has a Go repo with no requirement docs
        - When fitness runs
            - Then it fails
- 0071.6
    - Given Developer has an unchanged Go test without a requirement
        - When fitness runs on staged or pull request changes
            - Then it fails even for unrelated changes

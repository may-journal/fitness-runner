---
relatedConfigurations: ['../../.fitnessrc.json']
---

# 0065 Checks Requirement Tests

## Why

Each test proves one requirement in any language a module covers, so no acceptance is untested or tested twice.

## Measurement

acceptances owned by exactly one test
-
acceptances

Source: `go/cmd/fitness-check-requirements/owners.go:41`

## Requirements

- 0065.20
    - Given Developer has 0001.1 in a test name, or only in a comment, string, method, or `describe` title
        - When fitness runs
            - Then only the test the runner runs owns 0001.1
- 0065.21
    - Given Developer has an acceptance that no test owns
        - When fitness runs
            - Then it fails naming the test it needs
- 0065.17
    - Given Developer has an acceptance that two Go tests own
        - When fitness runs
            - Then it fails
- 0065.18
    - Given Developer has a Go test named for a removed acceptance
        - When fitness runs
            - Then it fails
- 0065.22
    - Given Developer has a test that proves no requirement
        - When fitness runs
            - Then it fails and asks for a name carrying an acceptance ID
- 0065.23
    - Given Developer has an unchanged test without a requirement
        - When fitness runs on staged or pull request changes
            - Then it fails even for unrelated changes
- 0065.24
    - Given Developer has code in a covered language and no requirement docs
        - When fitness runs
            - Then it fails
- 0065.25
    - Given Developer has no code in a covered language, or only a tool config such as `vite.config.ts`
        - When fitness runs
            - Then the check passes with nothing to judge

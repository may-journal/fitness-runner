---
relatedConfigurations: ['../../.fitnessrc.json']
---

# 0065 Checks Requirement Tests

## Why

A Go or Swift repo's tests prove its requirements, so each acceptance has exactly one test and nothing is untested or tested twice.

## Measurement

acceptances owned by exactly one Go or Swift test
-
acceptances

Source: `go/cmd/fitness-check-requirements/owners.go:100`

## Requirements

- 0065.19
    - Given Developer has `test0001_1` as an XCTest method, a `@Test func`, or only in a comment, string, or plain method
        - When fitness runs
            - Then only the XCTest method or `@Test func` owns 0001.1
- 0065.11
    - Given Developer has an acceptance that no Go or Swift test owns
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
- 0065.13
    - Given Developer has a Go or Swift test that proves no requirement
        - When fitness runs
            - Then it fails and asks for a name carrying an acceptance ID
- 0065.14
    - Given Developer has an unchanged Go or Swift test without a requirement
        - When fitness runs on staged or pull request changes
            - Then it fails even for unrelated changes
- 0065.15
    - Given Developer has a Go or Swift repo with tests and no requirement docs
        - When fitness runs
            - Then it fails
- 0065.16
    - Given Developer has a repo with no Go or Swift code
        - When fitness runs
            - Then the check passes with nothing to judge

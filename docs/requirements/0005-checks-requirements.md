---
relatedConfigurations: ['../../.fitnessrc.json']
---

# 0005 Checks Requirements

## Why

Every Go or Swift test proves a stated requirement and every requirement has exactly one test, so nothing is untested or tested twice.

## Measurement

acceptances owned by exactly one test
-
acceptances

Source: `go/cmd/fitness-check-requirements/owners.go:100`

## Requirements

- 0005.2
    - Given an acceptance that two tests own
        - When fitness runs
            - Then it fails
- 0005.8
    - Given an ID that an earlier commit deleted
        - When a doc brings it back
            - Then it fails
- 0005.16
    - Given a Go or Swift acceptance that no test owns
        - When fitness runs
            - Then it fails and names the test it needs
- 0005.17
    - Given a Go or Swift test named for no current acceptance
        - When fitness judges its file
            - Then it fails naming that test
- 0005.18
    - Given a Go or Swift repo with tests and no requirement docs
        - When fitness runs
            - Then it fails
- 0005.19
    - Given `test0001_1` in an `XCTestCase` class or as a Swift Testing `@Test`
        - When fitness runs
            - Then it owns 0001.1 and the check passes
- 0005.20
    - Given `test0001_1` in a comment, a string, or a plain method
        - When fitness runs
            - Then it is no test and 0001.1 fails
- 0005.21
    - Given a repo without Go or Swift code, or a scoped run changing no doc or test
        - When fitness runs
            - Then the check passes with nothing to judge

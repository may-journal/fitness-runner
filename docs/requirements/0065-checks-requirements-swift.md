---
relatedConfigurations: ['../../.fitnessrc.json']
---

# 0065 Checks Requirements Swift

## Why

A Swift app's tests prove its requirements the same way Go tests do, so each acceptance has exactly one test.

## Measurement

Swift acceptances owned by exactly one test
-
Swift acceptances

Source: `go/cmd/fitness-check-requirements/swift.go:57`

## Requirements

- 0065.1
    - Given Developer has an XCTest method `test0001_1` in an `XCTestCase` class
        - When fitness runs
            - Then it owns 0001.1 and the check passes
- 0065.2
    - Given Developer has a Swift Testing `@Test func test0001_1`
        - When fitness runs
            - Then it owns 0001.1 and the check passes
- 0065.3
    - Given Developer has a Swift repo where no test owns 0001.1
        - When fitness runs
            - Then it fails and names `test0001_1`
- 0065.4
    - Given Developer has a Swift test that proves no requirement
        - When fitness runs
            - Then it fails and asks for a `testNNNN_N` name
- 0065.5
    - Given Developer has `test0001_1` only inside a comment or a string
        - When fitness runs
            - Then it is no test and 0001.1 fails
- 0065.6
    - Given Developer has a `test0001_1` method outside any `XCTestCase` class and without `@Test`
        - When fitness runs
            - Then it is no test and 0001.1 fails
- 0065.7
    - Given Developer has a Swift repo with tests and no requirement docs
        - When fitness runs
            - Then it fails
- 0065.8
    - Given Developer has an unchanged Swift test without a requirement
        - When fitness runs on staged or pull request changes
            - Then it fails even for unrelated changes

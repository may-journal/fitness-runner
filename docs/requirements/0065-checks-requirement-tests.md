# 0065 Checks Requirement Tests

## Why

Each test proves one requirement in any language, so no acceptance is untested or tested twice, and Swift tests run inside the app.

## Measurement

acceptances owned by exactly one test
-
acceptances

## Requirements

- 0065.20
    - Given Developer has 0001.1 in a test name, or only in a comment, string, method, or `describe` title
        - When fitness runs
            - Then only the test the runner runs owns 0001.1
- 0065.26
    - Given Developer has an acceptance that no test or two tests own, or a test named for a removed acceptance
        - When fitness runs
            - Then it fails naming the acceptance and the test to add or remove
- 0065.27
    - Given Developer has a test that proves no requirement, changed or not
        - When fitness runs, even on unrelated staged or pull request changes
            - Then it fails and asks for a name carrying an acceptance ID
- 0065.24
    - Given Developer has code in a covered language and no requirement docs
        - When fitness runs
            - Then it fails
- 0065.25
    - Given Developer has no code in a covered language, or only a tool config such as `vite.config.ts`
        - When fitness runs
            - Then the check passes with nothing to judge
- 0065.28
    - Given Developer has a Swift test on XCTest, Xcode UI tests, simctl, another UI test framework, or no UITestingLibrary
        - When fitness runs
            - Then it fails naming the use and apple-ui-testing-library
- 0065.29
    - Given Developer has Swift Testing tests on UITestingLibrary, or end-to-end tests that drive the app through SystemSession
        - When fitness runs
            - Then the check passes

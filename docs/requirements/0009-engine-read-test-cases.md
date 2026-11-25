# 0009 Read Test Cases

## Why

My repo's own check reads each case a requirement test runs, so each step ties to the case proving it.

## Measurement

requirement tests whose named cases the shared scan lists

-

requirement tests

## Requirements

- 0009.1
  - Given Developer has a check of the repo's own that reads tests
    - When the repo holds a Go requirement test with `t.Run` cases and a case table
      - Then the check sees each case's name, file and line, and the step a name opens with
- 0009.2
  - Given Developer has a check of the repo's own that reads tests
    - When the repo holds a Java requirement test with dynamic tests and named arguments
      - Then the check sees each case's name, file and line, and the step a name opens with
- 0009.3
  - Given Developer has a check of the repo's own that reads tests
    - When the repo holds a Swift requirement test with named XCTest activities
      - Then the check sees each case's name, file and line, and the step a name opens with

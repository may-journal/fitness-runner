# 0001 Read Swift Test Cases

## Why

My repo's own check reads each case a Swift requirement test runs, so each step ties to the case proving it.

## Measurement

Swift requirement tests whose named cases the shared scan lists

-

Swift requirement tests

## Requirements

1. Reads Swift cases
   1. GIVEN Developer has a check of the repo's own that reads tests
      1. WHEN the repo holds a Swift requirement test with XCTest activities named by a fixed string
         1. THEN the check sees each activity's name
         2. THEN the check sees each activity's location
         3. THEN the check sees the step an activity name opens with

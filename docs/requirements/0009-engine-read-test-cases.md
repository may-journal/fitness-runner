# 0009 Read Test Cases

## Why

My repo's own check reads each case a requirement test runs, so each step ties to the case proving it.

## Measurement

requirement tests whose named cases the shared scan lists

-

requirement tests

## Requirements

1. Reads Go cases
   1. GIVEN Developer has a check of the repo's own that reads tests
      1. WHEN the repo holds a Go requirement test with `t.Run` cases plus case tables as slices or maps
         1. THEN the check sees each case's name
         2. THEN the check sees each case's location
         3. THEN the check sees the step a name opens with
2. Reads Java cases
   1. GIVEN Developer has a check of the repo's own that reads tests
      1. WHEN the repo holds a Java requirement test with dynamic tests plus `Named` arguments
         1. THEN the check sees each Java case's name
         2. THEN the check sees each Java case's location
         3. THEN the check sees the step a Java case name opens with
3. Reads Swift cases
   1. GIVEN Developer has a check of the repo's own that reads tests
      1. WHEN the repo holds a Swift requirement test with XCTest activities named by a fixed string
         1. THEN the check sees each activity's name
         2. THEN the check sees each activity's location
         3. THEN the check sees the step an activity name opens with

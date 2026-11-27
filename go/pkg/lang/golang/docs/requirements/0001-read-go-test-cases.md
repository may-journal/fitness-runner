# 0001 Read Go Test Cases

## Why

My repo's own check reads each case a Go requirement test runs, so each step ties to the case proving it.

## Measurement

Go requirement tests whose named cases the shared scan lists

-

Go requirement tests

## Requirements

1. Reads Go cases
   1. GIVEN Developer has a check of the repo's own that reads tests
      1. WHEN the repo holds a Go requirement test with `t.Run` cases plus case tables as slices or maps
         1. THEN the check sees each case's name
         2. THEN the check sees each case's location
         3. THEN the check sees the step a name opens with

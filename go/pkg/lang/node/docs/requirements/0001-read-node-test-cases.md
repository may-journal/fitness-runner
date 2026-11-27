# 0001 Read Node Test Cases

## Why

My repo's own check reads the groups and cases of each Node requirement test, so each step ties to the case proving it.

## Measurement

Node requirement tests whose named cases the shared scan lists

-

Node requirement tests

## Requirements

1. Reads Node cases
   1. GIVEN Developer has a check of the repo's own that reads tests
      1. WHEN the repo holds a Vitest or Jest requirement test whose case rows are array objects named by a fixed string
         1. THEN the check sees each case's name
         2. THEN the check sees each case's location
         3. THEN the check sees the step a case name opens with
2. Reads test groups and tables
   1. GIVEN Developer has a check of the repo's own that reads tests
      1. WHEN the repo holds a Vitest or Jest suite
         1. THEN the check sees each test's `describe` groups
         2. THEN each table is marked

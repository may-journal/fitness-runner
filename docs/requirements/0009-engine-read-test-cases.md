# 0009 Read Test Cases

## Why

My repo's own check reads each case a requirement test runs, so each step ties to the case proving it.

## Measurement

requirement tests whose named cases the shared scan lists

-

requirement tests

## Requirements

1. Removed in 2026.11.27.2030: moved to go/pkg/lang/golang as 0001.1
2. Removed in 2026.11.27.2030: moved to go/pkg/lang/java as 0001.1
3. Removed in 2026.11.27.2030: moved to go/pkg/lang/swift as 0001.1
4. Removed in 2026.11.27.2030: moved to go/pkg/lang/node as 0001.1
5. Lists each language's cases in one scan
   1. GIVEN Developer has a check of the repo's own that reads tests
      1. WHEN the repo holds requirement tests in two languages whose modules read cases
         1. THEN the shared scan lists each test's cases under it

# 0160 Checks engine free

## Why

Every check builds without the engine's GitHub reactions and run reports. So a check can later build and release on its own.

## Measurement

Checks whose Go dependencies hold no engine-only package
-
Checks

## Requirements

- 0160.1
    - Given Developer lists the Go dependencies of every check, tests included
        - When `go list -deps -test` runs
            - Then none holds `metadata`, `aftereffect`, `dupe` or `report`

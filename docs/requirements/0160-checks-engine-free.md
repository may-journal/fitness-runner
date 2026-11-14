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
            - Then it lists every check, none holds `metadata`, `aftereffect`, `dupe` or `report`, and a failure names the check and package
- 0160.2
    - Given Developer writes a Go module outside any workspace that imports every package under `go/pkg` from the working tree
        - When `go build` runs
            - Then it builds, since none of them is internal

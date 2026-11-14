# 0161 Engine check free

## Why

The engine links a check only through the generated built-ins map. So once the checks ship on their own, the engine builds without any of them.

## Measurement

Engine packages whose Go dependencies hold no check package
-
Engine packages

## Requirements

- 0161.1
    - Given Developer lists the Go dependencies of every engine file apart from the generated `checks.go`
        - When `go list -deps` runs
            - Then none is a check package, a failure names the importing package and the check, and `checks.go` still links the built-ins

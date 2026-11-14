# 0161 Engine check free

## Why

The engine links a check package only through the generated built-ins map. So removing that map leaves an engine that holds no check package.

## Measurement

Engine packages whose Go dependencies hold no check package
-
Engine packages

## Requirements

- 0161.1
    - Given Developer lists this module's Go dependencies of every engine file, for any platform, apart from the generated `checks.go`
        - When `go list -deps` runs
            - Then none is a check package, a failure names the importing package and the check, and `checks.go` still links the built-ins

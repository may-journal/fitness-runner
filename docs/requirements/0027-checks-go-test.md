---
relatedConfigurations: ['../../.fitnessrc.json']
---

# 0027 Checks Go Test

## Why

I never commit Go code whose own tests fail, and the check tells me which test broke and why.

## Measurement

Go modules whose tests pass
-
Go modules found

Source: `go/cmd/fitness-check-go-test/main.go:44`

## Requirements

- 0027.1
    - Given a Go test that fails with a message
        - When fitness runs go-test
            - Then it fails naming the test and its `file.go:line` message
- 0027.2
    - Given a Go module whose tests all pass
        - When fitness runs go-test
            - Then it passes having checked one module
- 0027.3
    - Given a repo with no `go.mod`
        - When fitness runs go-test
            - Then it passes having checked 0 files
- 0027.4
    - Given a broken Go module under `testdata` or `vendor`
        - When fitness runs go-test
            - Then it passes, since fixtures are not the repo's own code
- 0027.5
    - Given a failing Go module in a subfolder
        - When fitness runs go-test
            - Then each failure line starts with that folder's path
- 0027.6
    - Given a Go test file that does not compile
        - When fitness runs go-test
            - Then it fails quoting the build error at its `file.go:line`
- 0027.7
    - Given failing Go tests and only a non-Go file staged
        - When fitness runs go-test
            - Then it passes having checked 0 files
- 0027.8
    - Given a Go module and no Go toolchain on the path
        - When fitness runs go-test
            - Then it fails with a link to install Go

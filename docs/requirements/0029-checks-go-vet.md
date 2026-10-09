---
relatedConfigurations: ['../../.fitnessrc.json']
---

# 0029 Checks Go Vet

## Why

I catch suspicious Go code the compiler accepts before it ships. Repos without Go stay green, so the check is safe everywhere.

## Measurement

Go modules that `go vet` passes clean
-
Go modules found

Source: `go/cmd/fitness-check-go-vet/main.go:34`

## Requirements

- 0029.1
    - Given a Go module at the repo root with a vet problem
        - When fitness runs go-vet
            - Then it fails quoting the finding with its file and line
- 0029.2
    - Given a Go module in a subfolder with a vet problem
        - When fitness runs go-vet
            - Then the finding starts with the module folder
- 0029.3
    - Given a Go module with clean code
        - When fitness runs go-vet
            - Then it passes counting one module
- 0029.4
    - Given a repo with no `go.mod`
        - When fitness runs go-vet
            - Then it passes with 0 files
- 0029.5
    - Given a vet problem only in modules under `testdata` or `vendor`
        - When fitness runs go-vet
            - Then it passes with 0 files
- 0029.6
    - Given a module with a vet problem and a staged change to no Go file
        - When fitness runs go-vet on the staged change
            - Then it passes with 0 files
- 0029.7
    - Given a Go module and no Go toolchain on the path
        - When fitness runs go-vet
            - Then it fails with a one-line install hint

---
relatedConfigurations: ['../../.fitnessrc.json']
---

# 0028 Checks Go Test Coverage

## Why

I trust every command and library entry I ship, because its tests run each statement.

## Measurement

entry packages with every statement covered
-
entry packages measured

Source: `go/cmd/fitness-check-go-test-coverage/main.go:41`

## Requirements

- 0028.1
    - Given a command package with an untested statement
        - When go-test-coverage runs
            - Then it fails naming the entry and its statement count
- 0028.2
    - Given a command package whose tests run every statement
        - When go-test-coverage runs
            - Then it passes
- 0028.3
    - Given a library module with no command and no declared entry
        - When go-test-coverage runs
            - Then it fails asking for an `--entry` import path
- 0028.4
    - Given a library entry declared in `.fitnessrc.json` with an untested statement
        - When go-test-coverage runs
            - Then it fails naming the entry and its statement count
- 0028.5
    - Given an `--entry` flag naming a package no module has
        - When go-test-coverage runs
            - Then it fails naming the unknown entry
- 0028.6
    - Given a test that passes a named helper to `t.Run`
        - When go-test-coverage runs
            - Then it fails naming the test and the helper
- 0028.7
    - Given a repo without Go modules
        - When go-test-coverage runs
            - Then it passes with no files checked
- 0028.8
    - Given an `--audit-map` path to a new file outside the repo
        - When go-test-coverage runs
            - Then it passes and writes the source map there

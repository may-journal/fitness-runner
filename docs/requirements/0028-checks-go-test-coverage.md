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

## Requirements

- 0028.3
    - Given Developer has a library module with no command and no declared entry
        - When go-test-coverage runs
            - Then it fails asking for an `--entry` import path
- 0028.5
    - Given Developer has an `--entry` flag naming a package no module has
        - When go-test-coverage runs
            - Then it fails naming the unknown entry
- 0028.6
    - Given Developer has a test that passes a named helper to `t.Run`
        - When go-test-coverage runs
            - Then it fails naming the test and the helper
- 0028.7
    - Given Developer has a repo without Go modules
        - When go-test-coverage runs
            - Then it passes with no files checked
- 0028.8
    - Given Developer has an `--audit-map` path to a new file outside the repo
        - When go-test-coverage runs
            - Then it passes and writes the source map there
- 0028.9
    - Given Developer has a command and a declared library entry, each with an untested statement
        - When go-test-coverage runs
            - Then it fails naming each entry and its statement count
- 0028.10
    - Given Developer has a test whose coverage changes between runs
        - When go-test-coverage runs
            - Then it reports that test as unstable
- 0028.11
    - Given Developer has sixteen one-second tests, half parallel and half serial across packages
        - When go-test-coverage runs
            - Then it overlaps their runs exactly where go test would

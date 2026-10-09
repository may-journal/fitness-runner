---
relatedConfigurations: ['../../.fitnessrc.json']
---

# 0060 Checks Vitest Coverage Full

## Why

I trust every line of my JavaScript code ran under a test, because coverage below 100% turns the check red.

## Measurement

JavaScript projects whose Vitest coverage run passes at 100% thresholds
-
JavaScript projects checked

Source: `go/cmd/fitness-check-vitest-coverage-full/main.go:83`

## Requirements

- 0060.1
    - Given a repo with no `package.json`
        - When vitest-coverage-full runs
            - Then it passes having checked 0 files
- 0060.2
    - Given a Vitest config with a coverage threshold below 100
        - When vitest-coverage-full runs
            - Then it fails asking for thresholds of 100
- 0060.3
    - Given an installed shared config with a threshold below 100
        - When vitest-coverage-full runs
            - Then it fails asking the shared package for thresholds of 100
- 0060.4
    - Given fully tested code with Vitest installed and thresholds at 100
        - When vitest-coverage-full runs
            - Then it passes
- 0060.5
    - Given code that no test runs
        - When vitest-coverage-full runs
            - Then it fails showing the coverage Vitest reported
- 0060.6
    - Given a JavaScript project without Vitest installed
        - When vitest-coverage-full runs
            - Then it fails with a hint to install Vitest

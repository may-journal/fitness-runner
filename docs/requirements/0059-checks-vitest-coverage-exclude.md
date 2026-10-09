---
relatedConfigurations: ['../../.fitnessrc.json']
---

# 0059 Checks Vitest Coverage Exclude

## Why

I trust my coverage number, because Vitest measures every file and none hide behind an exclude list.

## Measurement

coverage exclude entries reported red
-
coverage exclude entries found

Source: `go/cmd/fitness-check-vitest-coverage-exclude/main.go:58`

## Requirements

- 0059.1
    - Given a repo without any `package.json`
        - When vitest-coverage-exclude runs
            - Then it passes having judged 0 files
- 0059.2
    - Given a Vitest config with an empty coverage exclude
        - When vitest-coverage-exclude runs
            - Then it passes
- 0059.3
    - Given a Vitest config that excludes test files and `dist`
        - When vitest-coverage-exclude runs
            - Then it fails naming each entry to remove
- 0059.4
    - Given a `package.json` whose `vitest` key excludes a file
        - When vitest-coverage-exclude runs
            - Then it fails naming that entry
- 0059.5
    - Given an empty Vitest config exclude and a `package.json` exclude
        - When vitest-coverage-exclude runs
            - Then it passes, since the config file wins
- 0059.6
    - Given an exclude with an imported name and a quoted path
        - When vitest-coverage-exclude runs
            - Then it fails naming only the quoted path
- 0059.7
    - Given no local Vitest config and an installed shared config with an exclude
        - When vitest-coverage-exclude runs
            - Then it fails naming the shared entry
- 0059.8
    - Given a `package.json` without any Vitest config or shared package
        - When vitest-coverage-exclude runs
            - Then it passes against the built-in shared config

---
relatedConfigurations: ['../../.fitnessrc.json']
---

# 0001 Runs Checks Reporting Status

## Why

Every check runs and ends green or red within its time budget, so I always know whether my code meets the bar.

## Measurement

runs whose exit code matches their checks' status
-
runs

Source: `go/cmd/fitness/ci.go:22`

## Requirements

- 0001.1
    - Given every check passes
        - When fitness runs
            - Then it reports green and exits 0
- 0001.2
    - Given a check that fails
        - When fitness runs
            - Then it reports red with that check's errors and exits 1
- 0001.3
    - Given a check that exits without a verdict
        - When fitness runs
            - Then it reports red and exits 1
- 0001.4
    - Given a check that runs past its time budget
        - When fitness runs
            - Then it is stopped and reported red as timed out
- 0001.5
    - Given a new check binary
        - When the runner lists its catalog
            - Then the new check is in it, so every repo runs it
- 0001.6
    - Given a repo that turns off one check by name
        - When the runner builds its list
            - Then only that check is left out
- 0001.7
    - Given a hung check that started a child process
        - When its time budget runs out
            - Then the child process is stopped too
- 0001.8
    - Given a repo that sets timeoutMs in its fitness config
        - When the runner budgets a check that declares none
            - Then the configured budget replaces the default

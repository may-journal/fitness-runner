---
relatedConfigurations: ['../../.fitnessrc.json']
---

# 0001 Runs Checks Reporting Status

## Why

Every check runs and ends green, red, or info within its time budget, so I always know whether my code meets the bar.

## Measurement

runs whose exit code matches their checks' status
-
runs

Source: `go/cmd/fitness/ci.go:22`

## Requirements

- 0001.1
    - Given Developer has every check passing
        - When fitness runs
            - Then it reports green and exits 0
- 0001.2
    - Given Developer has a check that fails
        - When fitness runs
            - Then it reports red with that check's errors and exits 1
- 0001.3
    - Given Developer has a check that panics, crashes, is killed, exits with an unexpected code, cannot start, or fails its describe step
        - When fitness runs
            - Then that check shows as info naming why, and the exit code is set by the other checks
- 0001.4
    - Given Developer has a check that runs past its time budget, or ignores the stop signal until its grace period runs out
        - When fitness runs
            - Then it and any child process it started are stopped, or killed if it ignores the stop, and reported red as timed out
- 0001.5
    - Given Developer has a new check binary
        - When the runner lists its catalog
            - Then the new check is in it and the run counts it, so every repo runs it
- 0001.8
    - Given Developer has a repo that sets timeoutMs in its fitness config
        - When the runner budgets a check that declares none
            - Then the configured budget replaces the default
- 0001.11
    - Given Developer has one real problem and a check that cannot run, such as `local-branch` with `gh` blocked
        - When fitness runs the full suite
            - Then the real problem fails, the other check shows as info, the run exits 1, and Actions annotates an error and a notice
- 0001.12
    - Given Developer has the same repo with the real problem fixed
        - When fitness runs
            - Then the check that cannot run shows as info and the run exits 0

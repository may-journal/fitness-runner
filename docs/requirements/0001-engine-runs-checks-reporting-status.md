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

## Requirements

- 0001.1
    - Given Developer has every check passing
        - When fitness runs
            - Then it reports every installed check green and exits 0
- 0001.2
    - Given Developer has a check that fails
        - When fitness runs
            - Then it reports red with that check's errors and exits 1
- 0001.5
    - Given Developer has a new check binary
        - When the runner lists its catalog
            - Then the new check is in it and the run counts it, so every repo runs it
- 0001.13
    - Given Developer has slow checks with different time budgets
        - When fitness runs
            - Then they overlap, the longest budget starts first, and the table keeps catalog order
- 0001.14
    - Given Developer has a check past the budget it declares or timeoutMs sets, or whose git or ssh waits for a password
        - When fitness runs
            - Then it and its child processes stop, or are killed if they ignore the stop, and it shows red as timed out
- 0001.15
    - Given Developer has a repo with tracked files, a few or thousands
        - When fitness runs every check
            - Then git lists the tracked files once for the whole run
- 0001.16
    - Given Developer has a tracked-file list from another repo, or from before the index changed
        - When a check walks the repo
            - Then it judges the files tracked now
- 0001.17
    - Given Developer has a check that panics, crashes, is killed, exits oddly, cannot start, or fails its describe step
        - When fitness runs
            - Then it shows as info naming why, and Actions adds a notice while the other checks set the exit code

# 0146 Run totals

## Why

No check fails my run on a limit of its own. The fast checks share one total, and when a run outgrows it, I see which checks to make faster.

## Measurement

runs past the fast total that fail and name the slowest checks with their times
-
runs past the fast total

## Requirements

- 0146.1
    - Given Developer has a check that runs past the fast total
        - When the run reaches that total
            - Then the run fails and names that check with its time
- 0146.2
    - Given Developer has a performance record that `fitness --json` kept
        - When `fitness perf` reads it
            - Then it lists each check's stage and time, slowest first
- 0146.3
    - Given Developer has a repo whose checks pass
        - When the run finishes
            - Then the summary names the slowest checks with their times

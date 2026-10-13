---
relatedConfigurations: ['../../.fitnessrc.json']
---

# 0083 Repo Check Settings

## Why

My repo adds, replaces, or turns off checks through the same command, hooks, and CI, and the org bar stays whole.

## Measurement

repo check settings that apply or name what stops them
-
repo check settings configured

Source: `go/cmd/fitness/local.go:34`

## Requirements

- 0083.1
    - Given Developer has a local check with a new name
        - When the full suite runs
            - Then the local check runs beside the bundled checks and reports its verdict
- 0083.2
    - Given Developer has external policy and a local check named like a bundled check
        - When the full suite runs
            - Then the local check runs in its place and the run names the replaced check
- 0083.3
    - Given Developer has org policy and a local check named like a bundled check
        - When the suite starts
            - Then it reports red naming the check and the org rule, and exits 1
- 0083.4
    - Given Developer has a check setting naming a directory or check that is not there
        - When the suite starts
            - Then it reports red naming the setting and exits 1
- 0083.5
    - Given Developer has a local check file without execute permission
        - When the suite starts
            - Then it reports red with the chmod command that fixes it and exits 1
- 0083.6
    - Given Developer has a local check with a new name
        - When fitness runs that check by name
            - Then only the local check runs
- 0083.7
    - Given Developer has external policy and disabledChecks naming a failing check
        - When the full suite runs
            - Then that check does not run and the rest report green
- 0083.8
    - Given Developer has org policy and disabledChecks naming a check
        - When the suite starts
            - Then it reports red naming the org rule and exits 1

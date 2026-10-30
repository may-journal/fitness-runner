# 0090 Repo Check Settings

## Why

My repo adds, replaces, or turns off checks through the same command, hooks, and CI, and the org bar stays whole.

## Measurement

repo check settings that apply or name what stops them
-
repo check settings configured

## Requirements

- 0090.1
    - Given Developer has a local check with a new name
        - When the full suite runs
            - Then the local check runs beside the bundled checks and reports its verdict
- 0090.2
    - Given Developer has external policy and a local check named like a bundled check
        - When the full suite or pr-check runs
            - Then the local check runs in its place and the run names the replaced check
- 0090.3
    - Given Developer has org policy and a local check named like a bundled check
        - When the suite starts
            - Then it reports red naming the check and the org rule, and exits 1
- 0090.4
    - Given Developer has a check setting whose directory or check is missing, outside the repo, or named like a subcommand
        - When the suite starts
            - Then it reports red naming the setting and exits 1
- 0090.5
    - Given Developer has a local check file without execute permission
        - When the suite starts
            - Then it reports red with the chmod command that fixes it and exits 1
- 0090.6
    - Given Developer has a local check with a new name
        - When fitness runs that check by name
            - Then only the local check runs
- 0090.7
    - Given Developer has external policy and disabledChecks naming a failing check
        - When the full suite or pr-check runs
            - Then that check does not run and the rest report green
- 0090.8
    - Given Developer has org policy and disabledChecks naming a check
        - When the suite starts
            - Then it reports red naming the org rule and exits 1

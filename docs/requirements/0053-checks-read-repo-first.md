---
relatedConfigurations: ['../../.fitnessrc.json']
---

# 0053 Checks Read Repo First

## Why

I see which checks guard this repo before I commit, so I read their rules instead of guessing.

## Measurement

runs that show the reminder and pass
-
runs of read-repo-first

Source: `go/cmd/fitness-check-read-repo-first/main.go:44`

## Requirements

- 0053.1
    - Given any repo
        - When fitness runs read-repo-first
            - Then it passes having judged 0 files
- 0053.2
    - Given any repo
        - When fitness runs read-repo-first
            - Then it asks whether I read the repo's decisions and enabled checks
- 0053.3
    - Given any repo
        - When fitness runs read-repo-first
            - Then it warns never to use `--no-verify`
- 0053.4
    - Given a run that enables only some checks
        - When read-repo-first prints its table
            - Then it lists exactly those checks beside their README paths
- 0053.5
    - Given a check name or path too long for its column
        - When read-repo-first prints its table
            - Then the cell is shortened, ending in `…`
- 0053.6
    - Given `NO_COLOR` is set
        - When fitness runs read-repo-first
            - Then its table prints with no color codes

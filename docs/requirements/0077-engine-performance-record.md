---
relatedConfigurations: ['../../.fitnessrc.json']
---

# 0077 Performance Record

## Why

Every run and release records what its checks and binaries cost, so I catch a slow check or growing download early.

## Measurement

runs asked for a record that write one with every check
-
runs asked for a record

Source: `go/cmd/fitness/main.go:208`

## Requirements

- 0077.1
    - Given Developer has a repo with passing and failing checks
        - When fitness runs with `--json=<file>`
            - Then the file lists every check's time, budget, files, binary size, and result
- 0077.2
    - Given Developer has a check that runs past its time budget
        - When fitness runs with `--json=<file>`
            - Then the file marks that check timed out
- 0077.3
    - Given Developer has a `--json` flag with no file, or a file it cannot write
        - When fitness runs with it
            - Then it names the problem and exits 1
- 0077.4
    - Given Developer has a run inside a GitHub Actions job
        - When fitness finishes
            - Then the job summary lists each check's time and budget, slowest first
- 0077.5
    - Given Developer has built release assets
        - When `fitness-release record` runs
            - Then it prints each asset's size and the install time as JSON

---
relatedConfigurations: ['../../.fitnessrc.json']
---

# 0077 Blob Input

## Why

I can ask whether a draft comment, issue body, or commit message would pass before I post it, without touching repo files.

## Measurement

blob runs that judge only the text and name every skipped check
-
blob runs

Source: `go/cmd/fitness/blob.go:43`

## Requirements

- 0077.1
    - Given Developer has a draft text and a repo with a file finding
        - When I run fitness with the text after `--blob`
            - Then every blob-capable check judges only the text, and each check that needs files is listed as skipped
- 0077.2
    - Given Developer has a draft text the chosen checks accept
        - When I run fitness `--blob` with `--checks` naming them and a check that needs files
            - Then only the chosen blob checks run and pass, the other is listed as skipped, and it exits 0
- 0077.3
    - Given Developer pipes a draft text with a finding on stdin
        - When I run fitness `--blob -`
            - Then the summary table names the finding on stderr and it exits 1
- 0077.4
    - Given Developer has an empty text, or adds `--all` or a check name
        - When I run fitness `--blob`
            - Then it names the usage error, runs no check, and exits 2
- 0077.5
    - Given Developer has a repo config that sets a check time budget
        - When I run fitness `--blob` on a draft text
            - Then each blob check runs within that budget, as in a normal run

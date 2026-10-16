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
            - Then the blob checks that apply judge only the text and pass, and each check that needs files is listed as skipped
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
- 0077.6
    - Given Developer has a Plan issue text that follows the Plan template
        - When I run fitness `--blob` on it with no `--checks`
            - Then the Plan checks pass, the PR and closing checks are listed as not applying, and it exits 0
- 0077.7
    - Given Developer has a PR description text that follows the PR template
        - When I run fitness `--blob` on it with no `--checks`
            - Then the PR checks pass, the Plan and closing checks are listed as not applying, and it exits 0
- 0077.8
    - Given Developer has a PR description text whose changelog section is empty
        - When I run fitness `--blob` on it with no `--checks`
            - Then pr-structure fails naming the section and it exits 1

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
    - Given Developer gives no text, selects no blob check, or adds `--all` or a check name
        - When I run fitness `--blob`
            - Then it names the usage error, runs no check, and exits 2
- 0077.6
    - Given Developer has a text that follows its template, such as a Plan whose steps name issues
        - When I run fitness `--blob` on it (a Plan issue, an Epic, or a PR description) with no `--checks`
            - Then its own checks pass, the comment and other body checks list in order as skipped, and it exits 0
- 0077.8
    - Given Developer has a PR description text whose changelog section is empty or its heading misnamed
        - When I run fitness `--blob` on it with no `--checks`
            - Then pr-structure fails naming the section and it exits 1
- 0077.9
    - Given Developer has a repo config that sets a check time budget or, under external policy, a checks list or disabledChecks
        - When I run fitness `--blob` on a draft text
            - Then each check keeps that budget and only the checks the config leaves on run, as in a normal run

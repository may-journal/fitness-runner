# 0101 Structured Findings

## Why

A failed run tells me where to change what, and how, without parsing prose. Each finding names the command that explains its check.

## Measurement

findings printed with `--format=json` that name a rule and an explain command
-
findings printed with `--format=json`

Source: `go/cmd/fitness/format.go:124`

## Requirements

- 0101.1
    - Given AI agent has a repo with a misspelled word and a broken link
        - When fitness runs with `--format=json`
            - Then stdout is one JSON object whose findings give each problem's file, line, rule, fix and explain command
- 0101.2
    - Given AI agent has a repo whose commit message breaks a check that reports no findings
        - When fitness runs with `--format=json`
            - Then that check's error lines come back as findings naming the check as their rule
- 0101.3
    - Given AI agent has a `--format` flag naming no known format
        - When fitness runs with it
            - Then it names the problem and exits 1
- 0101.4
    - Given AI agent has a repo with a misspelled word
        - When fitness runs in GitHub Actions
            - Then the annotation and the job summary give the finding's file, line, rule, fix and explain command, as `--format=json` does

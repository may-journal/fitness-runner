---
relatedConfigurations: ['../../.fitnessrc.json']
---

# 0014 Pull Requests

## Why

A pull request merges only when its title, description, and the issues it closes are all ready.

## Measurement

pull requests merged with a clean pr-check
-
pull requests merged

Source: `go/cmd/fitness/blobs.go:110`

## Requirements

- 0014.1
    - Given a PR that closes a finished Plan
        - When pr-check judges it
            - Then it reports green
- 0014.2
    - Given a PR whose title is not a semantic commit
        - When pr-check judges it
            - Then it reports red quoting the title
- 0014.3
    - Given a PR that closes a Plan with an unchecked item
        - When pr-check judges it
            - Then it reports red naming the item
- 0014.4
    - Given a PR that leaves open an issue its Plan closes
        - When pr-check judges it
            - Then it reports red naming that issue
- 0014.5
    - Given a manual workflow run
        - When pr-check runs
            - Then it judges every open PR

---
relatedConfigurations: ['../../.fitnessrc.json']
---

# 0050 Checks PR Closes Issue

## Why

Every pull request I merge closes the issues it delivers, so no finished work stays open.

## Measurement

PR bodies that close every issue they deliver
-
PR bodies checked

Source: `go/cmd/fitness-check-pr-closes-issue/main.go:92`

## Requirements

- 0050.1
    - Given a body that closes an issue with any of GitHub's nine closing keywords
        - When pr-closes-issue checks the body file
            - Then it passes
- 0050.2
    - Given a body that only references issues
        - When pr-closes-issue checks the body file
            - Then it fails saying it has no closing keyword
- 0050.3
    - Given a body that implements or plans an issue it does not close
        - When pr-closes-issue checks the body file
            - Then it fails naming each such issue
- 0050.4
    - Given a closing keyword only inside inline or fenced code
        - When pr-closes-issue checks the body file
            - Then it fails saying it has no closing keyword
- 0050.5
    - Given a required issue the body does not close
        - When pr-closes-issue checks it with `--require-close`
            - Then it fails naming that issue
- 0050.6
    - Given every required issue closed by the body
        - When pr-closes-issue checks it with `--require-close`
            - Then it passes
- 0050.7
    - Given no body at all
        - When pr-closes-issue runs
            - Then it passes having checked no files

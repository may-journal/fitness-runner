---
relatedConfigurations: ['../../.fitnessrc.json']
---

# 0009 Git Hooks

## Why

A commit or push that breaks the bar is stopped on my machine, before review or CI sees it. Every feature I push traces to a Plan a person approved.

## Measurement

bad commits and pushes the hooks refuse
-
bad commits and pushes tried

Source: `go/cmd/fitness/hook.go:15`

## Requirements

- 0009.1
    - Given a repo with no hooks
        - When I run fitness init
            - Then git runs the fitness commit-msg, pre-commit, and pre-push hooks
- 0009.3
    - Given the hooks and a message that is not a semantic commit
        - When I commit
            - Then the commit is refused naming semantic-commit
- 0009.4
    - Given the hooks and a failing executable pre-commit.local
        - When I commit a change that passes every check
            - Then the commit is refused
- 0009.7
    - Given a repo that already has a pre-commit hook
        - When I install the Go hook with --install-hook
            - Then the existing hook is kept and fitness reports red
- 0009.14
    - Given a branch I pushed
        - When I delete it from the remote
            - Then the deletion goes through
- 0009.15
    - Given a change that breaks a check, with hooks from fitness init or --install-hook
        - When I commit it
            - Then the commit is refused with that check's errors
- 0009.16
    - Given a feature commit whose trailer names an approved Plan, a non-Plan issue, an unapproved Plan, or nothing
        - When I push it
            - Then only the approved Plan goes through, and the rest are refused naming why
- 0009.17
    - Given only chore commits, main's feature commits merged in, or a branch whose first push named an approved Plan
        - When I push it
            - Then the push goes through

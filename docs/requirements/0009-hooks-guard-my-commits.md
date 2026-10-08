---
relatedConfigurations: ['../../.fitnessrc.json']
---

# 0009 Hooks Guard My Commits

## Why

A commit or push that breaks the bar is stopped on my machine, before review or CI sees it.

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
- 0009.2
    - Given the hooks and a change that breaks a check
        - When I commit it
            - Then the commit is refused with that check's errors
- 0009.3
    - Given the hooks and a message that is not a semantic commit
        - When I commit
            - Then the commit is refused naming semantic-commit
- 0009.4
    - Given the hooks and a failing executable pre-commit.local
        - When I commit a change that passes every check
            - Then the commit is refused
- 0009.5
    - Given the hooks and a feature commit with no Plan
        - When I push it
            - Then the push is refused asking for a Plan trailer
- 0009.6
    - Given the hooks and only chore or docs commits
        - When I push them
            - Then the push goes through
- 0009.7
    - Given a repo that already has a pre-commit hook
        - When I install the Go hook with --install-hook
            - Then the existing hook is kept and fitness reports red

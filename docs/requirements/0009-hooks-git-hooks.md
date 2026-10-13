---
relatedConfigurations: ['../../.fitnessrc.json']
---

# 0009 Git Hooks

## Why

A commit or push that breaks the bar is stopped on my machine, before review or CI sees it.

## Measurement

bad commits and pushes the hooks refuse
-
bad commits and pushes tried

Source: `go/cmd/fitness/hook.go:15`

## Requirements

- 0009.1
    - Given Developer has a repo with no hooks
        - When I run fitness init
            - Then git runs the fitness commit-msg, pre-commit, and pre-push hooks
- 0009.2
    - Given Developer has the hooks and a change that breaks a check
        - When I commit it
            - Then the commit is refused with that check's errors
- 0009.3
    - Given Developer has the hooks and a message that is not a semantic commit
        - When I commit
            - Then the commit is refused naming semantic-commit
- 0009.4
    - Given Developer has the hooks and a failing executable pre-commit.local
        - When I commit a change that passes every check
            - Then the commit is refused
- 0009.7
    - Given Developer has a repo that already has a pre-commit hook
        - When I install the Go hook with --install-hook
            - Then the existing hook is kept and fitness reports red
- 0009.8
    - Given Developer has the Go hook installed with --install-hook and a GitHub remote
        - When I commit a change that breaks a check
            - Then the commit is refused with that check's errors without reading PR metadata
- 0009.9
    - Given Developer has the hooks, external policy, and disabledChecks naming semantic-commit
        - When I commit a message that is not a semantic commit
            - Then the commit goes through, since the hook skips the disabled check

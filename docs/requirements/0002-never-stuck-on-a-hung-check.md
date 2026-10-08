---
relatedConfigurations: ['../../.fitnessrc.json']
---

# 0002 Never Stuck on a Hung Check

## Why

A hung check cannot leave processes running on the machine, and each repo can set how long checks may take.

## Measurement

overrunning checks stopped with every process they started
-
checks that overran their time budget

Source: `go/cmd/fitness/timeout.go:12`

## Requirements

- 0002.1
    - Given a hung check that started a child process
        - When its time budget runs out
            - Then the child process is stopped too
- 0002.2
    - Given a repo that sets timeoutMs in its fitness config
        - When the runner budgets a check that declares none
            - Then the configured budget replaces the default

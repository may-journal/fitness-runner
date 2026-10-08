---
relatedConfigurations: ['../../.fitnessrc.json']
---

# 0002 Never Stuck on a Hung Check

## Why

A hung check cannot freeze my commit, my agent, or my CI job, or leave processes running on the machine.

## Measurement

overrunning checks stopped with every process they started
-
checks that overran their time budget

Source: `go/cmd/fitness/main.go:632`

## Requirements

- 0002.1
    - Given a check that runs past its time budget
        - When the runner runs it
            - Then it is stopped and reported as timed out
- 0002.2
    - Given a hung check that started a child process
        - When its time budget runs out
            - Then the child process is stopped too

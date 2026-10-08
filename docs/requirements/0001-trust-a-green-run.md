---
relatedConfigurations: ['../../.fitnessrc.json']
---

# 0001 Trust a Green Run

## Why

When fitness passes, I can merge without wondering what it skipped or swallowed.

## Measurement

green runs where every check returned a verdict
-
green runs

Source: `go/cmd/fitness/main.go:648`

## Requirements

- 0001.1
    - Given a check that exits without a verdict
        - When the runner runs it
            - Then the whole run fails

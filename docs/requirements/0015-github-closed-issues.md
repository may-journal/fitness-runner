---
relatedConfigurations: ['../../.fitnessrc.json']
---

# 0015 Closed Issues

## Why

An issue closed with unfinished work comes back open, so nothing is quietly dropped.

## Measurement

issues closed with unchecked items that are reopened
-
issues closed as completed with unchecked items

Source: `go/internal/aftereffect/reopen.go:12`

## Requirements

- 0015.1
    - Given an issue closed as completed with an unchecked item
        - When close-check runs
            - Then it reopens the issue with a comment naming the item and the closer
- 0015.2
    - Given a close that close-check already handled
        - When it runs again
            - Then it adds no second comment
- 0015.3
    - Given an issue closed as not planned with unchecked items
        - When close-check runs
            - Then the issue stays closed
- 0015.4
    - Given an issue closed with every item ticked
        - When close-check runs
            - Then the issue stays closed

---
relatedConfigurations: ['../../.fitnessrc.json']
---

# 0012 Checks Issue Checklist

## Why

An issue with unfinished items is never closed as done, so its open work stays visible.

## Measurement

issue bodies with unchecked items reported red
-
issue bodies with unchecked items

Source: `go/cmd/fitness-check-issue-checklist/main.go:28`

## Requirements

- 0012.1
    - Given an issue body with an unchecked item
        - When fitness runs issue-checklist on it
            - Then it reports red quoting the item
- 0012.2
    - Given an issue body with every item ticked
        - When fitness runs issue-checklist on it
            - Then it reports green

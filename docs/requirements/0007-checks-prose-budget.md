---
relatedConfigurations: ['../../.fitnessrc.json']
---

# 0007 Checks Prose Budget

## Why

I can read any doc, plan, or PR in a minute, because no sentence, paragraph, or list runs on.

## Measurement

docs within every prose limit
-
docs checked

Source: `go/cmd/fitness-check-prose-budget/main.go:120`

## Requirements

- 0007.1
    - Given a sentence, paragraph, or list past its limit
        - When prose-budget checks the doc
            - Then it fails naming the limit
- 0007.2
    - Given a section past its word budget
        - When prose-budget checks the doc
            - Then it fails naming the section
- 0007.3
    - Given a CHANGELOG past a limit
        - When prose-budget runs
            - Then it is judged like any other doc
- 0007.4
    - Given an issue or PR body past a limit
        - When prose-budget checks it in body mode
            - Then it fails like a file would
- 0007.5
    - Given a nested list item with more than half its parent level cap
        - When prose-budget checks the doc
            - Then it fails naming the level

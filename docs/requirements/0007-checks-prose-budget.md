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
    - Given a sentence past its word limit
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
    - Given a list with more items than its level allows
        - When prose-budget checks the doc
            - Then it fails naming the level
- 0007.6
    - Given a paragraph past its sentence limit
        - When prose-budget checks the doc
            - Then it fails naming the limit
- 0007.7
    - Given long lines inside a fenced code block
        - When prose-budget checks the doc
            - Then it passes, since code is not prose
- 0007.8
    - Given a repo that sets its own proseBudget limits
        - When prose-budget checks the doc
            - Then it judges against those limits

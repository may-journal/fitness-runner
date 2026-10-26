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

Source: `go/cmd/fitness-check-prose-budget/main.go:110`

## Requirements

- 0007.2
  - Given Developer has a section past its word budget
    - When prose-budget checks the doc
      - Then it fails naming the section
- 0007.3
  - Given Developer has a CHANGELOG past a limit
    - When prose-budget runs
      - Then it is judged like any other doc
- 0007.4
  - Given Developer has an issue or PR body past a limit
    - When prose-budget checks it in body mode
      - Then it fails like a file would
- 0007.5
  - Given Developer has a list with more items than its level allows
    - When prose-budget checks the doc
      - Then it fails naming the level
- 0007.7
  - Given Developer has a fenced code block, or a Measurement whose `-` line sits between blank lines
    - When prose-budget checks the doc
      - Then it passes, since neither is prose
- 0007.8
  - Given Developer has a repo that sets its own proseBudget limits
    - When prose-budget checks the doc
      - Then it judges against those limits
- 0007.9
  - Given Developer has a sentence past its word limit, or a paragraph past its sentence limit
    - When prose-budget checks the doc
      - Then it fails naming the limit
- 0007.10
  - Given Developer has a ticked Plan task within its word limit
    - When prose-budget checks the Plan
      - Then it passes, since the box is not a word

---
relatedConfigurations: ['../../.fitnessrc.json']
---

# 0039 Checks Mermaid Callouts

## Why

I can match every numbered callout in a diagram to exactly one row explaining it.

## Measurement

docs whose numbered diagrams match their callout tables one to one
-
docs with a mermaid diagram or callout table

Source: `go/cmd/fitness-check-mermaid-callouts/main.go:115`

## Requirements

- 0039.1
    - Given a doc whose diagram and table callouts match one to one
        - When mermaid-callouts checks the repo
            - Then it passes, counting only that doc
- 0039.2
    - Given a diagram callout with no table row
        - When mermaid-callouts checks the repo
            - Then it fails naming the callout and diagram line
- 0039.3
    - Given a table row with no diagram callout
        - When mermaid-callouts checks the repo
            - Then it fails naming the row and table line
- 0039.4
    - Given a callout number repeated in a diagram or table
        - When mermaid-callouts checks the repo
            - Then it fails naming the number and its count
- 0039.5
    - Given a numbered diagram with no callout table
        - When mermaid-callouts checks the repo
            - Then it fails naming the diagram line
- 0039.6
    - Given a callout table with no diagram before it
        - When mermaid-callouts checks the repo
            - Then it fails naming the table line
- 0039.7
    - Given an un-numbered legend diagram between a diagram and its table
        - When mermaid-callouts checks the repo
            - Then it passes, since un-numbered diagrams need no table
- 0039.8
    - Given an issue or PR body with a diagram callout missing its row
        - When mermaid-callouts checks it in body mode
            - Then it fails like a file would

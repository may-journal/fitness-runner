---
relatedConfigurations: ['../../.fitnessrc.json']
---

# 0039 Checks Mermaid Callouts

## Why

I can match every callout in a diagram to exactly one row explaining it. Nodes carry numbers and connectors carry letters, so I tell them apart at a glance.

## Measurement

docs whose diagrams match their callout tables one to one
-
docs with a mermaid diagram or callout table

## Requirements

- 0039.1
    - Given Developer has a doc whose callouts match its table one to one, beside prose labels such as YES and HTTPS
        - When mermaid-callouts checks the repo
            - Then it passes, counting only that doc
- 0039.9
    - Given Developer has a callout in only the diagram or only its table
        - When mermaid-callouts checks the repo
            - Then it fails naming each unmatched callout and its line
- 0039.4
    - Given Developer has a callout repeated in a diagram or table
        - When mermaid-callouts checks the repo
            - Then it fails naming the callout and its count
- 0039.10
    - Given Developer has a diagram with callouts but no table, or a table with no diagram
        - When mermaid-callouts checks the repo
            - Then it fails naming the unpaired block's line
- 0039.7
    - Given Developer has an un-numbered legend diagram between a diagram and its table
        - When mermaid-callouts checks the repo
            - Then it passes, since un-numbered diagrams need no table
- 0039.8
    - Given Developer has an issue or PR body with a diagram callout missing its row
        - When mermaid-callouts checks it in body mode
            - Then it fails like a file would
- 0039.11
    - Given Developer has a connector labelled with a number
        - When mermaid-callouts checks the repo
            - Then it fails asking for a letter
- 0039.12
    - Given Developer has a connector letter that breaks the doubling, such as AB after Z
        - When mermaid-callouts checks the repo
            - Then it fails naming the letter in the diagram and table

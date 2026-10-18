---
relatedConfigurations: ['../../.fitnessrc.json']
---

# 0084 Checks Mermaid Syntax

## Why

Every mermaid diagram I publish renders, because a broken one fails before anyone reads it.

## Measurement

mermaid diagrams that parse
-
mermaid diagrams

Source: `go/cmd/fitness-check-mermaid-syntax/main.go:93`

## Requirements

- 0084.1
    - Given Developer has valid flowcharts, graphs, sequence diagrams, and other types, with inline edge labels, description blocks, and shapes that leave brackets unpaired
        - When mermaid-syntax checks the doc
            - Then it passes
- 0084.2
    - Given Developer has a flowchart node label with unquoted brackets
        - When mermaid-syntax checks the doc
            - Then it fails naming the node and its line
- 0084.3
    - Given Developer has a flowchart with an edge missing its target and an unclosed subgraph
        - When mermaid-syntax checks the doc
            - Then it fails naming each line
- 0084.4
    - Given Developer has a sequence diagram with a message missing its colon and an unclosed loop
        - When mermaid-syntax checks the doc
            - Then it fails naming each line
- 0084.5
    - Given Developer has a diagram of an unknown type
        - When mermaid-syntax checks the doc
            - Then it fails naming the type
- 0084.6
    - Given Developer has a C4 diagram with an unclosed bracket
        - When mermaid-syntax checks the doc
            - Then it fails naming the bracket's line
- 0084.7
    - Given Developer has a PR body with a broken flowchart
        - When mermaid-syntax checks it in body mode
            - Then it fails like a file would
- 0084.8
    - Given Developer has docs with no mermaid diagrams
        - When mermaid-syntax runs
            - Then it passes having checked no files

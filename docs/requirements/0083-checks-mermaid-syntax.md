---
relatedConfigurations: ['../../.fitnessrc.json']
---

# 0083 Checks Mermaid Syntax

## Why

Every mermaid diagram I publish renders, because a broken one fails before anyone reads it.

## Measurement

mermaid diagrams that parse
-
mermaid diagrams

Source: `go/cmd/fitness-check-mermaid-syntax/main.go:93`

## Requirements

- 0083.1
    - Given Developer has a valid flowchart, graph, sequence diagram, and C4 diagram
        - When mermaid-syntax checks the doc
            - Then it passes
- 0083.2
    - Given Developer has a flowchart node label with unquoted brackets
        - When mermaid-syntax checks the doc
            - Then it fails naming the node and its line
- 0083.3
    - Given Developer has a flowchart with an edge missing its target and an unclosed subgraph
        - When mermaid-syntax checks the doc
            - Then it fails naming each line
- 0083.4
    - Given Developer has a sequence diagram with a message missing its colon and an unclosed loop
        - When mermaid-syntax checks the doc
            - Then it fails naming each line
- 0083.5
    - Given Developer has a diagram of an unknown type
        - When mermaid-syntax checks the doc
            - Then it fails naming the type
- 0083.6
    - Given Developer has a C4 diagram with an unclosed bracket
        - When mermaid-syntax checks the doc
            - Then it fails naming the bracket's line
- 0083.7
    - Given Developer has a PR body with a broken flowchart
        - When mermaid-syntax checks it in body mode
            - Then it fails like a file would
- 0083.8
    - Given Developer has docs with no mermaid diagrams
        - When mermaid-syntax runs
            - Then it passes having checked no files

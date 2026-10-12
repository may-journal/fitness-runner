---
relatedConfigurations: ['../../.fitnessrc.json']
---

# 0042 Checks Mermaid Legend

## Why

I can match every numbered callout in a flowchart to its legend, because each one is styled and the legend exists.

## Measurement

numbered flowchart callouts with a style class
-
numbered flowchart callouts

Source: `go/cmd/fitness-check-mermaid-legend/main.go:140`

## Requirements

- 0042.1
    - Given Developer has a flowchart callout node with no style class
        - When mermaid-legend checks the doc
            - Then it fails naming the callout and its diagram line
- 0042.2
    - Given Developer has a flowchart with numbered callouts but no `classDef` legend
        - When mermaid-legend checks the doc
            - Then it fails saying the legend is missing
- 0042.3
    - Given Developer has callouts styled inline, by `class`, or by `style`, with a legend
        - When mermaid-legend checks the doc
            - Then it passes
- 0042.4
    - Given Developer has a diagram that is not a flowchart
        - When mermaid-legend checks the doc
            - Then it passes, since other diagrams style differently
- 0042.5
    - Given Developer has flowchart labels that do not open with a separate number
        - When mermaid-legend checks the doc
            - Then it passes, since they are not callouts
- 0042.6
    - Given Developer has a doc with a clean diagram and then a broken one
        - When mermaid-legend checks the doc
            - Then it names the broken diagram's own line
- 0042.7
    - Given Developer has a PR body with an unstyled callout
        - When mermaid-legend checks it in body mode
            - Then it fails like a file would
- 0042.8
    - Given Developer has docs with no mermaid diagrams
        - When mermaid-legend runs
            - Then it passes having checked no files

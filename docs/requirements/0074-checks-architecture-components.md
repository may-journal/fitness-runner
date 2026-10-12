---
relatedConfigurations: ['../../.fitnessrc.json']
---

# 0074 Architecture Components

## Why

Every product folder has one component owner, and calls between components follow the diagram. A new folder or a call the diagram does not draw fails.

## Measurement

component views whose owners, edges, and runtimes match the sources
-
component views checked

Source: #288

## Requirements

- 0074.1
    - Given Developer has no architecture
        - When I check its components
            - Then it passes with zero files
- 0074.2
    - Given Developer has a product source no component names
        - When I check its components
            - Then the error names the source, and naming its folder passes
- 0074.3
    - Given Developer has one source two components in different diagrams name
        - When I check its components
            - Then the error names both claims
- 0074.4
    - Given Developer has a Go import or Swift reference the view does not draw
        - When I check its components
            - Then the error names the reference line, the view, and a reversed edge
- 0074.5
    - Given Developer has an edge row that links evidence
        - When I check its components
            - Then the link must be a source of the first component that names the second, outside comments
- 0074.6
    - Given Developer has a component under one container that another runtime reaches
        - When I check its components
            - Then the error names the reference and the container, and a component outside every container passes
- 0074.7
    - Given Developer has a reference under a build condition
        - When I check its components
            - Then a missing edge fails naming the condition, and a drawn edge passes
- 0074.8
    - Given Developer has unchanged component drift
        - When I check unrelated changes
            - Then the same findings remain

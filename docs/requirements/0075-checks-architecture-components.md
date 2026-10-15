---
relatedConfigurations: ['../../.fitnessrc.json']
---

# 0075 Architecture Components

## Why

Every product folder has one component owner, and calls between components follow the diagram. A new folder or a call the diagram does not draw fails.

## Measurement

component views whose owners, edges, and runtimes match the sources
-
component views checked

Source: #288

## Requirements

- 0075.1
    - Given Developer has no architecture
        - When I check its components
            - Then it passes with zero files
- 0075.3
    - Given Developer has one source two components in different diagrams name
        - When I check its components
            - Then the error names both claims
- 0075.5
    - Given Developer has an edge row that links evidence
        - When I check its components
            - Then the link must be a source of the first component that names the second, outside comments
- 0075.6
    - Given Developer has a component under one container that another runtime reaches
        - When I check its components
            - Then the error names the reference and the container, and a component outside every container passes
- 0075.11
    - Given Developer has Swift overloads, extension members, and local names in function bodies
        - When I check its components
            - Then each name reaches the declaration its labels and base type pick, and a local name reaches nothing
- 0075.12
    - Given Developer has a product source no component names
        - When I check its components
            - Then the error names the source, naming its folder or root source passes, and root tool configs need no owner
- 0075.13
    - Given Developer has a Go import or Swift reference the view does not draw, also under a build condition
        - When I check its components
            - Then the error names the reference line, the view, a reversed edge, and the condition, and a drawn edge passes
- 0075.14
    - Given Developer has a script only package.json scripts run, or one under `scripts/`
        - When I check its components
            - Then it needs no owner, and a file the start script runs or the package publishes still does

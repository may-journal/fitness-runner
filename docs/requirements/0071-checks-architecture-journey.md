---
relatedConfigurations: ['../../.fitnessrc.json']
---

# 0071 Checks Architecture Journey

## Why

Every requirement has a place in the user journey, and every journey step that promises behavior names a real requirement. Neither side drifts while the checks stay green.

## Measurement

requirement docs with one journey row
-
requirement docs

Source: #290

## Requirements

- 0071.1
    - Given Developer has a journey table with one row per requirement doc
        - When I check it
            - Then it passes, with several nodes per doc and several docs per node
- 0071.2
    - Given Developer has a requirement doc with no journey row
        - When I check the journey
            - Then the error names the table line and the doc ID
- 0071.3
    - Given Developer has a row naming an acceptance ID, a deleted doc, or an unknown doc
        - When I check the journey
            - Then the error names the row line and that ID
- 0071.4
    - Given Developer has two rows for one requirement doc
        - When I check the journey
            - Then the error names both row lines and the doc ID
- 0071.5
    - Given Developer has a row listing no node or a node the journey does not declare
        - When I check the journey
            - Then the error names the row line, the doc ID, and the node
- 0071.6
    - Given Developer has a screen, action, or system node in no row
        - When I check the journey
            - Then the error names its diagram line, while personas, subgraphs, and legends need no row
- 0071.7
    - Given Developer has a renamed node label or a renamed node ID
        - When I check the journey
            - Then the label passes and the ID fails its stale row and its unlisted node
- 0071.8
    - Given Developer has drift on one side only
        - When I check changes on the other side
            - Then the same findings remain

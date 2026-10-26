---
relatedConfigurations: ['../../.fitnessrc.json']
---

# 0072 Checks Architecture Journey

## Why

Every requirement appears in the user journey diagram at the step that fulfils it, and every step that promises behavior names one. Neither side drifts while the checks stay green.

## Measurement

requirement docs named in a journey diagram label
-
requirement docs

## Requirements

- 0072.1
    - Given Developer has a journey diagram whose labels name every requirement doc ID
        - When I check it
            - Then it passes with several IDs per label or one ID on several steps, and Mermaid codes like `#9829;` name none
- 0072.2
    - Given Developer has a requirement doc that no journey label names
        - When I check the journey
            - Then the error names the diagram line and the doc ID
- 0072.3
    - Given Developer has a label naming an acceptance ID, a deleted doc, or an unknown doc
        - When I check the journey
            - Then the error names the label line and that ID, with no second error for that step
- 0072.4
    - Given Developer has a label naming more than eight requirement IDs
        - When I check the journey
            - Then the error names the label line and asks to split the step
- 0072.5
    - Given Developer has requirement IDs only in a table or prose outside the diagram
        - When I check the journey
            - Then they cover nothing, and each doc is still missing from the diagram
- 0072.6
    - Given Developer has a screen, action, or system step with no ID in its label or its subgraph label
        - When I check the journey
            - Then the error names its diagram line, while personas, subgraphs, and legends need no ID
- 0072.7
    - Given Developer has a renamed node ID or label text that keeps its IDs
        - When I check the journey
            - Then it passes, and a label edit that drops an ID fails for that doc
- 0072.8
    - Given Developer has drift on one side only
        - When I check changes on the other side
            - Then the same findings remain

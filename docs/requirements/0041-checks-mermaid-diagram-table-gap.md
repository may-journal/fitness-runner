---
relatedConfigurations: ['../../.fitnessrc.json']
---

# 0041 Checks Mermaid Diagram Table Gap

## Why

I read every diagram detail in its callout table, so no paragraph between diagram and table repeats or contradicts it.

## Measurement

diagram-to-table gaps holding at most the caption line
-
diagram-to-table gaps checked

## Requirements

- 0041.1
    - Given Developer has a paragraph between a numbered diagram and its callout table
        - When mermaid-diagram-table-gap checks the doc
            - Then it fails naming the file and line
- 0041.2
    - Given Developer has a paragraph between the diagram's legend and its callout table
        - When mermaid-diagram-table-gap checks the doc
            - Then it fails naming that line
- 0041.3
    - Given Developer has only a caption saying the numbers match the callout table
        - When mermaid-diagram-table-gap checks the doc
            - Then it passes, whatever the caption's middle wording or trailing clause
- 0041.4
    - Given Developer has a caption followed by another paragraph before the table
        - When mermaid-diagram-table-gap checks the doc
            - Then it fails on the paragraph but not the caption
- 0041.5
    - Given Developer has prose before a diagram, after its table, or around an unnumbered diagram
        - When mermaid-diagram-table-gap checks the doc
            - Then it passes
- 0041.6
    - Given Developer has markdown with no diagram and no callout table
        - When mermaid-diagram-table-gap runs
            - Then it passes having checked 0 files
- 0041.7
    - Given Developer has a PR body with a paragraph between diagram and table
        - When mermaid-diagram-table-gap checks it in body mode
            - Then it fails naming the description

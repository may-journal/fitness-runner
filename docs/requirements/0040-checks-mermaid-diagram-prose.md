# 0040 Checks Mermaid Diagram Prose

## Why

I read each arrow's meaning once, in the callout table, so a diagram never repeats or contradicts it.

## Measurement

paired diagrams whose relationship labels hold only callout numbers
-
paired diagrams checked

## Requirements

- 0040.1
    - Given Developer has a paired diagram whose `Rel` label has prose after its number or letter
        - When mermaid-diagram-prose checks the doc
            - Then it fails quoting the label
- 0040.2
    - Given Developer has a paired flowchart whose pipe edge label has prose after its number or letter
        - When mermaid-diagram-prose checks the doc
            - Then it fails quoting the label
- 0040.3
    - Given Developer has a paired diagram whose colon edge label has prose after its number or letter
        - When mermaid-diagram-prose checks the doc
            - Then it fails quoting the label
- 0040.4
    - Given Developer has a paired diagram with only bare number labels, comments, and style colors
        - When mermaid-diagram-prose checks the doc
            - Then it passes
- 0040.5
    - Given Developer has a diagram with prose labels but no callout table
        - When mermaid-diagram-prose checks the doc
            - Then it passes
- 0040.6
    - Given Developer has a paired diagram whose node names carry prose
        - When mermaid-diagram-prose checks the doc
            - Then it passes, since only arrows are judged
- 0040.7
    - Given Developer has a PR body whose paired diagram has a prose label
        - When mermaid-diagram-prose checks it in body mode
            - Then it fails like a file would
- 0040.8
    - Given Developer has a repo with no diagram or callout table
        - When mermaid-diagram-prose runs
            - Then it passes having judged 0 files

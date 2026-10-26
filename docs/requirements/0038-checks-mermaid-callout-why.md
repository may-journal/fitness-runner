# 0038 Checks Mermaid Callout Why

## Why

I learn why each diagram element exists, not just what it is, because every callout row explains its reason.

## Measurement

numbered callout rows with a filled Why cell
-
numbered callout rows checked

## Requirements

- 0038.1
    - Given Developer has a callout table without a Why column
        - When mermaid-callout-why checks the doc
            - Then it fails naming the file and table line
- 0038.2
    - Given Developer has a numbered callout row with an empty Why cell
        - When mermaid-callout-why checks the doc
            - Then it fails naming the row number as written
- 0038.3
    - Given Developer has a numbered callout row that ends before the Why column
        - When mermaid-callout-why checks the doc
            - Then it fails as if the cell were empty
- 0038.4
    - Given Developer has a `WHY` header and a reason in every numbered row
        - When mermaid-callout-why checks the doc
            - Then it passes
- 0038.5
    - Given Developer has an unnumbered callout row with an empty Why cell
        - When mermaid-callout-why checks the doc
            - Then it passes, since only numbered rows need a reason
- 0038.6
    - Given Developer has a callout table without a Why column and no diagram
        - When mermaid-callout-why checks the doc
            - Then it still fails naming the table line
- 0038.7
    - Given Developer has docs with no diagram and only plain tables
        - When mermaid-callout-why runs
            - Then it passes having judged no files
- 0038.8
    - Given Developer has an issue or PR body with a callout table lacking a Why column
        - When mermaid-callout-why checks it in body mode
            - Then it fails like a file would

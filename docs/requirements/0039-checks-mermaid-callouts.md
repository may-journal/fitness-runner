# 0039 Checks Mermaid Callouts

## Why

I can match every callout in a diagram to exactly one row explaining it. Nodes carry numbers and connectors carry letters, so I tell them apart at a glance.

## Measurement

docs whose diagrams match their callout tables one to one
-
docs with a mermaid diagram or callout table

## Requirements

- 0039.14
    - Given Developer has callouts matching their table one to one, beside prose such as YES and HTTPS and a legend diagram
        - When mermaid-callouts checks the repo
            - Then it passes, counting only that doc
- 0039.13
    - Given Developer has a callout in only the diagram or only its table, a repeated callout, or an unpaired diagram or table
        - When mermaid-callouts checks the repo
            - Then it fails naming each unmatched callout and its line, each repeated callout and its count, and each unpaired block's line
- 0039.8
    - Given Developer has an issue or PR body with a diagram callout missing its row
        - When mermaid-callouts checks it in body mode
            - Then it fails like a file would
- 0039.11
    - Given Developer has a connector labelled with a number
        - When mermaid-callouts checks the repo
            - Then it fails asking for a letter and naming `fitness mermaid-callouts --fix`
- 0039.12
    - Given Developer has a connector letter that breaks the doubling, such as AB after Z
        - When mermaid-callouts checks the repo
            - Then it fails naming the letter in the diagram and table
- 0039.15
    - Given Developer has connectors labelled with numbers beside numbered nodes and a letter already in use
        - When mermaid-callouts runs with --fix
            - Then each connector and its table row get the next free letter, the file is listed, and the check passes
- 0039.16
    - Given Developer has converted the docs with --fix
        - When mermaid-callouts runs with --fix again
            - Then it changes no file and lists none
- 0039.17
    - Given Developer has a numbered connector whose number also marks a node
        - When mermaid-callouts runs with --fix
            - Then it names the diagram and why, leaves the file byte for byte, and still fails

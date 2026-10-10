---
relatedConfigurations: ['../../.fitnessrc.json']
---

# 0066 Checks Doc Template Choices

## Why

One folder can hold several kinds of doc, each with its own template, beside it or in a `templates` subfolder. Every doc there passes by following any one of them.

## Measurement

docs that match one of their folder's templates
-
docs in folders with several templates

Source: `go/cmd/fitness-check-doc-template/main.go:119`

## Requirements

- 0066.1
    - Given a folder with two templates and a doc matching the second
        - When doc-template checks the repo
            - Then it passes
- 0066.2
    - Given a folder with two templates and a doc matching neither
        - When doc-template checks the repo
            - Then it fails naming the closest template and its differences
- 0066.3
    - Given a doc matching one of the files in its folder's `templates` subfolder
        - When doc-template checks the repo
            - Then it passes
- 0066.4
    - Given a doc in a subfolder with its own template, under a `templates` subfolder's folder
        - When doc-template checks the repo
            - Then it is judged against its own folder's template
- 0066.5
    - Given a file in a `templates` subfolder with a section lacking a guiding comment
        - When doc-template checks the repo
            - Then it fails naming the section

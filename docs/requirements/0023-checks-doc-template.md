---
relatedConfigurations: ['../../.fitnessrc.json']
---

# 0023 Checks Doc Template

## Why

I write a template once per kind of doc, and every doc beside it follows one of them. Each template section explains what to write, so authors are never guessing.

## Measurement

markdown files that match their template, and templates that guide every section
-
markdown files checked

Source: `go/cmd/fitness-check-doc-template/main.go:61`

## Requirements

- 0023.12
    - Given Developer has docs matching one of their folder's templates, beside a GitHub template they do not match
        - When doc-template checks the repo
            - Then it passes, since GitHub templates govern bodies, not files
- 0023.10
    - Given Developer has a doc with a section its template lacks, or without one it has
        - When doc-template checks the repo
            - Then it fails naming each such section
- 0023.4
    - Given Developer has a doc in a subfolder under nested templates, or beside named templates that are not its namesake
        - When doc-template checks the repo
            - Then it is judged against the nearest folder's templates above it that govern it
- 0023.11
    - Given Developer has a template, or a GitHub pull request or issue template, with a section lacking a guiding comment
        - When doc-template checks the repo
            - Then it fails naming the section
- 0023.9
    - Given Developer has a folder with two templates and a doc matching neither
        - When doc-template checks the repo
            - Then it fails naming both templates
- 0023.13
    - Given Developer has views copied from the shipped architecture templates, with sections of their own and no front matter, beside other docs
        - When doc-template checks the repo
            - Then it passes
- 0023.14
    - Given Developer has a view missing a legend or opening section its template has
        - When doc-template checks the repo
            - Then it fails naming what the template has
- 0023.15
    - Given Developer has a view whose callout table lacks its template's columns or caption
        - When doc-template checks the repo
            - Then it fails quoting the template's columns or caption

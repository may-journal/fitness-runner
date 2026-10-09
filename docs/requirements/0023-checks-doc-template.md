---
relatedConfigurations: ['../../.fitnessrc.json']
---

# 0023 Checks Doc Template

## Why

I write a template once, and every doc beside it keeps the same sections. Each template section explains what to write, so authors are never guessing.

## Measurement

markdown files that match their template, and templates that guide every section
-
markdown files checked

Source: `go/cmd/fitness-check-doc-template/main.go:117`

## Requirements

- 0023.1
    - Given a doc whose sections match its folder's `template.md`
        - When doc-template checks the repo
            - Then it passes
- 0023.2
    - Given a doc with a section its template lacks
        - When doc-template checks the repo
            - Then it fails naming the unexpected section
- 0023.3
    - Given a doc missing a section its template has
        - When doc-template checks the repo
            - Then it fails naming the missing section
- 0023.4
    - Given a doc in a subfolder under nested templates
        - When doc-template checks the repo
            - Then it is judged against the nearest template above it
- 0023.5
    - Given docs with no template above them
        - When doc-template checks the repo
            - Then it passes whatever their sections
- 0023.6
    - Given a template section without a guiding comment
        - When doc-template checks the repo
            - Then it fails naming the section
- 0023.7
    - Given a GitHub pull request or issue template section without a guiding comment
        - When doc-template checks the repo
            - Then it fails naming the section
- 0023.8
    - Given a GitHub template and a sibling doc with other sections
        - When doc-template checks the repo
            - Then it passes, since GitHub templates govern bodies, not files

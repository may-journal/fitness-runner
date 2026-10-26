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

Source: `go/cmd/fitness-check-doc-template/main.go:101`

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
- 0023.16
    - Given Developer has a view that lacks a legend, opening section, callout column or caption its template has
        - When doc-template checks the repo
            - Then it fails naming or quoting what the template has
- 0023.17
    - Given Developer has views started from the shipped architecture templates, with or without a copy, and sections of their own
        - When doc-template checks the repo
            - Then it passes, and a view that drifts from a shipped template warns until opted in, then fails
- 0023.18
    - Given Developer has a view whose Mermaid type, direction, style, color, or element kind differs from its template's
        - When doc-template checks the repo
            - Then it warns until opted in, then fails, naming what the template uses; a bad opt-in value fails

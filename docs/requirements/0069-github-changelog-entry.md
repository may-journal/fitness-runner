---
relatedConfigurations: ['../../.fitnessrc.json']
---

# 0069 Changelog Entry

## Why

Each pull request adds one dated changelog entry that describes the whole change, however many commits it has.

## Measurement

pull requests that add exactly one dated changelog entry
-
pull requests that change the changelog

Source: `go/internal/metadata/changelog.go:16`

## Requirements

- 0069.1
    - Given a PR whose later commits amend its one dated entry
        - When pr-check and branch-check count its changelog entries
            - Then both pass
- 0069.2
    - Given a PR whose commits add two dated entries
        - When pr-check and branch-check count its changelog entries
            - Then both fail naming the extra heading
- 0069.3
    - Given a PR that changes the changelog without a dated entry
        - When pr-check counts its changelog entries
            - Then it fails asking for one entry
- 0069.4
    - Given a release PR that adds a version section
        - When pr-check counts its changelog entries
            - Then it passes

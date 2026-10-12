---
relatedConfigurations: ['../../.fitnessrc.json']
---

# 0069 Changelog Entry

## Why

Each pull request adds one dated changelog entry that describes the whole change, however many commits it has. Its description links that entry instead of copying it.

## Measurement

pull requests that add exactly one dated changelog entry and link it
-
pull requests that change the changelog

Source: `go/internal/metadata/changelog.go:17`

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
    - Given a release PR that adds a version section and links it
        - When pr-check judges its changelog entry
            - Then it passes
- 0069.5
    - Given a PR description that links a different changelog entry
        - When pr-check judges its changelog entry
            - Then it fails naming the entry the PR adds
- 0069.6
    - Given a PR description that copies a bullet of its changelog entry
        - When pr-check judges its changelog entry
            - Then it fails quoting the copied line
- 0069.7
    - Given a release PR description that copies its release notes
        - When the release tool rewrites it
            - Then it links the changelog section and compare view instead
- 0069.8
    - Given a PR description with no link to the changelog entry it adds
        - When pr-check judges its changelog entry
            - Then it fails naming the entry to link

---
relatedConfigurations: ['../../.fitnessrc.json']
---

# 0069 Changelog Entry

## Why

Each pull request adds one dated changelog entry that describes the whole change, however many commits it has. Its description links that entry instead of copying it. As its own entry file, the preferred style, it never conflicts with another open pull request.

## Measurement

pull requests that add exactly one dated changelog entry and link it
-
pull requests that change the changelog

Source: `go/internal/metadata/changelog.go:17`

## Requirements

- 0069.4
    - Given Developer has a release PR that adds a version section and links it
        - When pr-check judges its changelog entry
            - Then it passes
- 0069.6
    - Given Developer has a PR description that copies a bullet of its changelog entry
        - When pr-check judges its changelog entry
            - Then it fails quoting the copied line
- 0069.7
    - Given Developer has a release PR description that copies its release notes
        - When the release tool rewrites it
            - Then it links the changelog section and compare view instead
- 0069.9
    - Given Developer has a PR description that links a different changelog entry, or none
        - When pr-check judges its changelog entry
            - Then it fails naming the entry to link
- 0069.10
    - Given Developer has a PR whose later commits amend its one entry, a `CHANGELOG.md` section or an entry file
        - When pr-check and branch-check count its changelog entries
            - Then both pass
- 0069.11
    - Given Developer has a PR whose commits add two entries, as sections, entry files, or one of each
        - When pr-check and branch-check count its changelog entries
            - Then both fail naming the extra one
- 0069.12
    - Given Developer has a PR that changes the changelog without adding an entry
        - When pr-check counts its changelog entries
            - Then it fails asking for one, suggesting an entry file
- 0069.13
    - Given Developer has entry files waiting for the open release PR
        - When the release tool rewrites it
            - Then it commits them under the version section and deletes the files

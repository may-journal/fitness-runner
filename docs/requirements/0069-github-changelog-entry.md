---
relatedConfigurations: ['../../.fitnessrc.json']
---

# 0069 Changelog Entry

## Why

Each pull request adds one dated changelog entry file that describes the whole change, however many commits it has. Its description links that file instead of copying it. As its own file, it never conflicts with another open pull request.

## Measurement

pull requests that add exactly one dated changelog entry file and link it
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
- 0069.13
    - Given Developer has entry files on main, and the open release PR's branch holds them or no entry folder
        - When the release tool rewrites it
            - Then it commits the ones the branch holds under the version section and deletes those files
- 0069.14
    - Given Developer has a PR that amends its own or a pending entry, or adds the first `CHANGELOG.md`, even when rebased
        - When pr-check and branch-check count its changelog entries
            - Then both pass
- 0069.15
    - Given Developer has a PR whose commits add two entry files
        - When pr-check and branch-check count its changelog entries
            - Then both fail naming the extra one
- 0069.16
    - Given Developer has a PR that adds or edits a section in an existing `CHANGELOG.md`
        - When pr-check and branch-check judge its changelog entry
            - Then both fail pointing to an entry file in `changelog/`

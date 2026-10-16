---
relatedConfigurations: ['../../.fitnessrc.json']
---

# 0069 Changelog Entry

## Why

Each pull request adds one dated changelog entry file that describes the whole change, however many commits it has. Its description links that file instead of copying it. As its own file, it never conflicts with another open pull request.

Entry files are the only changelog. Each tagged release lists the entries added since the one before, so the notes live where people read them.

## Measurement

pull requests that add exactly one dated changelog entry file and link it
-
pull requests that change the changelog

Source: `go/internal/metadata/changelog.go:17`

## Requirements

- 0069.17
    - Given Developer has a PR description that links a different changelog entry or none, or copies one of its bullets
        - When pr-check judges its changelog entry
            - Then it fails naming the entry to link or quoting the copied line
- 0069.14
    - Given Developer has a PR that amends its own or a pending entry, or adds the first `CHANGELOG.md`, even when rebased
        - When pr-check and branch-check count its changelog entries
            - Then both pass
- 0069.15
    - Given Developer has a PR whose commits add two entry files
        - When pr-check and branch-check count its changelog entries
            - Then both fail naming the extra one
- 0069.19
    - Given Developer has a PR that adds an entry file whose name lacks its heading's date
        - When pr-check and branch-check judge its changelog entry
            - Then both fail naming the dated file name they expect
- 0069.20
    - Given Developer has a release PR that adds no entry file, or a PR that edits `CHANGELOG.md` beside its linked entry file
        - When pr-check and branch-check judge its changelog entry
            - Then both pass
- 0069.21
    - Given Developer has a release PR description that copies its release notes
        - When the release tool rewrites it
            - Then it links the entry files and compare view instead, and commits nothing
- 0069.22
    - Given Developer has entry files from before the previous release tag and entry files added since
        - When the release tool writes the new tag's release notes
            - Then the description lists only the new entries, newest first, and links the compare view

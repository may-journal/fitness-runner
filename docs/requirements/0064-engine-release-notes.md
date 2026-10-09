---
relatedConfigurations: ['../../.fitnessrc.json']
---

# 0064 Release Notes In The House Format

## Why

A release PR passes the changelog checks without hand edits, because its notes arrive in the format the checks expect.

## Measurement

release PRs whose notes pass the changelog checks untouched
-
release PRs

Source: `go/internal/release/notes.go:39`

## Requirements

- 0064.1
    - Given a changelog whose newest release Release Please wrote
        - When I run fitness-release house-changelog
            - Then that release becomes a dated version heading over typed bullets that keep their links
- 0064.2
    - Given Features, Bug Fixes, and Performance Improvements subheadings
        - When I run fitness-release house-changelog
            - Then their bullets become Feat, Fix, and Perf bullets
- 0064.3
    - Given timestamped entries and older releases around the new release
        - When I run fitness-release house-changelog
            - Then those lines stay exactly as they were
- 0064.4
    - Given a changelog whose newest release is already in the house format
        - When I run fitness-release house-changelog
            - Then the file is left unchanged
- 0064.5
    - Given a release subheading the house format has no type for
        - When I run fitness-release house-changelog
            - Then it fails naming that subheading
- 0064.6
    - Given a changelog the command has rewritten
        - When the changelog and changelog-bullets checks run
            - Then they pass

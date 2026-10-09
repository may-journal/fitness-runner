---
relatedConfigurations: ['../../.fitnessrc.json']
---

# 0054 Checks Release Changelog

## Why

I approve a release only when its version file and changelog section match the version Release Please will tag.

## Measurement

release versions whose version file and changelog section agree
-
release versions in a manifest

Source: `go/cmd/fitness-check-release-changelog/main.go:79`

## Requirements

- 0054.1
    - Given a repo with no Release Please manifest
        - When fitness runs release-changelog
            - Then it passes judging 0 files
- 0054.2
    - Given an empty manifest during bootstrap
        - When fitness runs release-changelog
            - Then it passes
- 0054.3
    - Given a matching `version.txt` and a `## [1.2.3] (date)` section with a list item
        - When fitness runs release-changelog
            - Then it passes
- 0054.4
    - Given a manifest that lacks a numeric root `.` version
        - When fitness runs release-changelog
            - Then it fails naming what is wrong with the manifest
- 0054.5
    - Given a `version.txt` that differs from the manifest
        - When fitness runs release-changelog
            - Then it fails naming both versions
- 0054.6
    - Given a release with no `version.txt` and no CHANGELOG
        - When fitness runs release-changelog
            - Then it fails naming each missing file
- 0054.7
    - Given a release section that is missing, empty, or duplicated
        - When fitness runs release-changelog
            - Then it fails naming the section problem
- 0054.8
    - Given a staged manifest change without a CHANGELOG change
        - When fitness runs release-changelog
            - Then it fails asking for the release section

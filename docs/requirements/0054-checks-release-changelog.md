---
relatedConfigurations: ['../../.fitnessrc.json']
---

# 0054 Checks Release Changelog

## Why

I approve a release only when its version file matches the version Release Please will tag. Its notes come from entry files, so my `CHANGELOG.md` needs no release section.

## Measurement

release versions whose version file agrees
-
release versions in a manifest

Source: `go/cmd/fitness-check-release-changelog/main.go:71`

## Requirements

- 0054.1
    - Given Developer has a repo with no Release Please manifest
        - When fitness runs release-changelog
            - Then it passes judging 0 files
- 0054.2
    - Given Developer has an empty manifest during bootstrap
        - When fitness runs release-changelog
            - Then it passes
- 0054.4
    - Given Developer has a manifest that lacks a numeric root `.` version
        - When fitness runs release-changelog
            - Then it fails naming what is wrong with the manifest
- 0054.5
    - Given Developer has a `version.txt` that differs from the manifest
        - When fitness runs release-changelog
            - Then it fails naming both versions
- 0054.11
    - Given Developer has a release with no `version.txt`
        - When fitness runs release-changelog
            - Then it fails naming the missing file
- 0054.12
    - Given Developer has a matching `version.txt`, a staged manifest change, and a `CHANGELOG.md` with no release section, or none
        - When fitness runs release-changelog
            - Then it passes

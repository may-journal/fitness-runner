---
relatedConfigurations: ['../../.fitnessrc.json']
---

# 0017 Checks Changelog

## Why

I can tell which release each change shipped in, because every changelog section is dated and matches the package version. Release notes written by Release Please pass as they are.

## Measurement

changelogs with dated sections that match the package version
-
changelogs checked

Source: `go/cmd/fitness-check-changelog/main.go:63`

## Requirements

- 0017.1
    - Given a repo with no root `CHANGELOG.md`
        - When changelog runs
            - Then it fails saying the changelog is missing
- 0017.2
    - Given a changelog with no `###` section
        - When changelog runs
            - Then it fails asking for a dated section
- 0017.5
    - Given a `package.json` version suffix that differs from the first heading
        - When changelog runs
            - Then it fails naming the mismatch
- 0017.6
    - Given a `package-lock.json` version that differs from `package.json`
        - When changelog runs
            - Then it fails naming the lock mismatch
- 0017.7
    - Given a `package.json` or `package-lock.json` that is not valid JSON
        - When changelog runs
            - Then it fails naming the invalid file
- 0017.8
    - Given package files whose versions match the first heading
        - When changelog runs
            - Then it passes
- 0017.11
    - Given dated `###` headings, some with notes, and a release section's Features and Bug Fixes
        - When changelog runs
            - Then it passes
- 0017.12
    - Given an undated `###` heading, among the changes or outside any release section
        - When changelog runs
            - Then it fails quoting that heading

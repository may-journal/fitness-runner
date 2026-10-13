---
relatedConfigurations: ['../../.fitnessrc.json']
---

# 0017 Checks Changelog

## Why

I can tell which release each change shipped in, because every changelog section is dated and matches the package version. Release notes written by Release Please pass as they are, and so do entry files waiting for a release.

## Measurement

changelogs with dated sections that match the package version
-
changelogs checked

Source: `go/cmd/fitness-check-changelog/main.go:63`

## Requirements

- 0017.1
    - Given Developer has a repo with no root `CHANGELOG.md`
        - When changelog runs
            - Then it fails saying the changelog is missing
- 0017.2
    - Given Developer has a changelog with no `###` section
        - When changelog runs
            - Then it fails asking for a dated section
- 0017.5
    - Given Developer has a `package.json` version suffix that differs from the first heading
        - When changelog runs
            - Then it fails naming the mismatch
- 0017.8
    - Given Developer has package files whose versions match the first heading
        - When changelog runs
            - Then it passes
- 0017.12
    - Given Developer has an undated `###` heading, among the changes or outside any release section
        - When changelog runs
            - Then it fails quoting that heading
- 0017.14
    - Given Developer has a `package-lock.json` version that differs from `package.json`, or a package file that is not valid JSON
        - When changelog runs
            - Then it fails naming that file
- 0017.15
    - Given Developer has dated `###` sections and a release section in `CHANGELOG.md`, with or without entry files
        - When changelog runs
            - Then it passes, reading the newest entry of either style first
- 0017.16
    - Given Developer has an entry file without exactly one dated `###` heading
        - When changelog runs
            - Then it fails naming that file

---
relatedConfigurations: ['../../.fitnessrc.json']
---

# 0017 Checks Changelog

## Why

I can tell when each change was made, because every entry file holds one dated section and carries that date in its name. My `CHANGELOG.md` is mine to shape, so an upstream fork's history passes as it is.

## Measurement

entry files with one dated section and a dated name
-
entry files checked

## Requirements

- 0017.21
    - Given Developer has an entry file whose name lacks its heading's date or carries another
        - When changelog runs
            - Then it fails naming the file and the dated name it expects
- 0017.22
    - Given Developer has an entry file with two dated headings, or with an undated one
        - When changelog runs
            - Then it fails naming that file
- 0017.23
    - Given Developer has a `CHANGELOG.md` with upstream `### Added` sections and an unstamped `package.json` version, beside valid entry files with and without front matter
        - When changelog runs
            - Then it passes, judging only the entry files
- 0017.24
    - Given Developer has a repo with no entry files, with or without a `CHANGELOG.md`
        - When changelog runs
            - Then it passes with no files checked

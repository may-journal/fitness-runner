---
relatedConfigurations: ['../../.fitnessrc.json']
---

# 0018 Checks Changelog Bullets

## Why

I can skim every release's notes, because each entry file is a few short bullets, each starting with its kind of change. My `CHANGELOG.md` is not judged, so an upstream fork's history passes as it is.

## Measurement

entry files within every bullet rule
-
entry files checked

Source: `go/cmd/fitness-check-changelog-bullets/main.go:32`

## Requirements

- 0018.12
    - Given Developer has a bullet of 365 characters or more, even wrapped across lines
        - When changelog-bullets runs
            - Then it fails naming the bullet's first line and whole length
- 0018.16
    - Given Developer has an entry with fewer than 3 or more than 5 bullets, counting only dash bullets
        - When changelog-bullets runs
            - Then it fails naming the section and its bullet count
- 0018.17
    - Given Developer has an entry file with two bullets or an untyped one
        - When changelog-bullets runs
            - Then it fails naming that file
- 0018.18
    - Given Developer has a `CHANGELOG.md` of upstream numbered `### Added` lists and no entry files
        - When changelog-bullets runs
            - Then it passes with no files checked
- 0018.19
    - Given Developer has an entry file with bullets starting with `Feat:`, a subject like `fix(release)!:`, or no type
        - When changelog-bullets runs
            - Then only the untyped bullet fails, asking for a type

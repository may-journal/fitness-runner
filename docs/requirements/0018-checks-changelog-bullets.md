---
relatedConfigurations: ['../../.fitnessrc.json']
---

# 0018 Checks Changelog Bullets

## Why

I can skim every changelog release, because each entry is a few short bullets, each starting with its kind of change. Release notes written by Release Please pass as they are.

## Measurement

changelog sections within every bullet rule
-
changelog sections checked

Source: `go/cmd/fitness-check-changelog-bullets/main.go:70`

## Requirements

- 0018.1
    - Given Developer has a changelog section with fewer than 3 or more than 5 bullets
        - When changelog-bullets runs
            - Then it fails naming the section and its bullet count
- 0018.6
    - Given Developer has an older section that breaks the rules
        - When changelog-bullets runs
            - Then it fails, since history meets the bar too
- 0018.7
    - Given Developer has a repo without a `CHANGELOG.md`
        - When changelog-bullets runs
            - Then it passes with no files checked
- 0018.9
    - Given Developer has Release Please bullets with a bold scope and no type prefix
        - When changelog-bullets and markdown-no-bold-italic run
            - Then they pass
- 0018.11
    - Given Developer has a timestamped entry using a star bullet
        - When changelog-bullets runs
            - Then the entry still fails as having too few bullets
- 0018.12
    - Given Developer has a bullet of 365 characters or more, even wrapped across lines
        - When changelog-bullets runs
            - Then it fails naming the bullet's first line and whole length
- 0018.13
    - Given Developer has bullets starting with `Feat:`, a subject like `fix(release)!:`, or no type
        - When changelog-bullets runs
            - Then only the untyped bullet fails, asking for a type
- 0018.14
    - Given Developer has a valid entry above a release over the configured list limit, even across subheadings
        - When changelog-bullets runs
            - Then only the release fails, naming that limit

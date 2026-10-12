---
relatedConfigurations: ['../../.fitnessrc.json']
---

# 0018 Checks Changelog Bullets

## Why

I can skim every changelog release, because each entry is a few short bullets, each starting with its kind of change.

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
- 0018.2
    - Given Developer has a bullet of 365 characters or more
        - When changelog-bullets runs
            - Then it fails naming the bullet's line and length
- 0018.3
    - Given Developer has a long bullet wrapped across several lines
        - When changelog-bullets runs
            - Then it counts the whole bullet and names its first line
- 0018.4
    - Given Developer has a bullet without a semantic type prefix
        - When changelog-bullets runs
            - Then it fails asking for a type such as `Feat:`
- 0018.5
    - Given Developer has bullets starting with `Feat:` or a commit subject like `fix(release)!:`
        - When changelog-bullets runs
            - Then it passes
- 0018.6
    - Given Developer has an older section that breaks the rules
        - When changelog-bullets runs
            - Then it fails, since history meets the bar too
- 0018.7
    - Given Developer has a repo without a `CHANGELOG.md`
        - When changelog-bullets runs
            - Then it passes with no files checked
- 0018.8
    - Given Developer has an entry above a release section with more bullets than `proseBudget.maxListItems`
        - When changelog-bullets runs
            - Then only the release section fails, naming that limit

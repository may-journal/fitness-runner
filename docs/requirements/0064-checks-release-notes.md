---
relatedConfigurations: ['../../.fitnessrc.json']
---

# 0064 Checks Release Notes

## Why

Release PRs pass the changelog checks as Release Please writes them, so nobody rewrites release notes by hand.

## Measurement

release PRs whose notes pass the changelog checks untouched
-
release PRs

Source: `go/cmd/fitness-check-changelog/main.go:88`

## Requirements

- 0064.1
    - Given a release section with Features and Bug Fixes subheadings
        - When the changelog check runs
            - Then it passes
- 0064.2
    - Given an undated ### heading outside any release section
        - When the changelog check runs
            - Then it fails naming that heading
- 0064.3
    - Given Release Please bullets with a bold scope and no type prefix
        - When changelog-bullets and markdown-no-bold-italic run
            - Then they pass
- 0064.4
    - Given bold text in a release section that is not a bullet's scope
        - When markdown-no-bold-italic runs
            - Then it fails quoting that bold
- 0064.5
    - Given a release with more bullets across its subheadings than the list limit
        - When changelog-bullets runs
            - Then it fails naming the release and the limit
- 0064.6
    - Given a timestamped entry using a star bullet
        - When changelog-bullets runs
            - Then the entry still fails as having too few bullets
- 0064.7
    - Given a release section under Release Please's linked version heading
        - When the release-changelog check runs
            - Then it finds the section and passes

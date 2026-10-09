---
relatedConfigurations: ['../../.fitnessrc.json']
---

# 0019 Checks Changelog Updated

## Why

Every commit I make carries a dated changelog entry that names what changed. I can trust each heading time, because no one guesses it.

## Measurement

staged commits whose changelog entry is current and names the change
-
staged commits with changes

Source: `go/cmd/fitness-check-changelog-updated/main.go:284`

## Requirements

- 0019.1
    - Given a current entry naming the change, above older re-added sections
        - When changelog-updated checks the staged commit
            - Then it passes
- 0019.2
    - Given staged changes without a staged changelog entry
        - When changelog-updated checks the staged commit
            - Then it fails asking me to stage `CHANGELOG.md`
- 0019.3
    - Given an entry sharing fewer than three words with the change
        - When changelog-updated checks the staged commit
            - Then it fails with the count and suggested words
- 0019.4
    - Given a newest heading older than five minutes
        - When changelog-updated checks the staged commit
            - Then it fails naming the expected heading time
- 0019.5
    - Given a `package.json` version ending in a different time
        - When changelog-updated checks the staged commit
            - Then it fails expecting the version's time
- 0019.6
    - Given a merge in progress that replays an old entry
        - When changelog-updated checks the staged commit
            - Then it passes without judging the old heading
- 0019.7
    - Given nothing staged
        - When changelog-updated runs
            - Then it passes with no files checked
- 0019.8
    - Given staged changes and no `CHANGELOG.md` on disk
        - When changelog-updated checks the staged commit
            - Then it fails asking me to add it

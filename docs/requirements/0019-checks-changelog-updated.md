---
relatedConfigurations: ['../../.fitnessrc.json']
---

# 0019 Checks Changelog Updated

## Why

My branch carries one dated changelog entry, in its own entry file. Each commit adds bullets to it that name what changed. I can trust its heading time, because no one guesses it.

## Measurement

staged commits whose changelog bullets name the change under the branch's one current entry
-
staged commits with changes

Source: `go/cmd/fitness-check-changelog-updated/main.go:105`

## Requirements

- 0019.3
    - Given Developer has an entry sharing fewer than three words with the change
        - When changelog-updated checks the staged commit
            - Then it fails with the count and suggested words
- 0019.4
    - Given Developer has a newest heading older than five minutes
        - When changelog-updated checks the staged commit
            - Then it fails naming the expected heading time
- 0019.5
    - Given Developer has a `package.json` version ending in a different time
        - When changelog-updated checks the staged commit
            - Then it fails expecting the version's time
- 0019.6
    - Given Developer has a merge in progress that replays an old entry
        - When changelog-updated checks the staged commit
            - Then it passes without judging the old heading
- 0019.13
    - Given Developer has staged changes without a staged entry file, or no `CHANGELOG.md`
        - When changelog-updated checks the staged commit
            - Then it fails asking me to add an entry file
- 0019.11
    - Given Developer has staged a new entry file naming the change
        - When changelog-updated checks the staged commit
            - Then it passes without a `CHANGELOG.md` change
- 0019.14
    - Given Developer has a branch whose earlier commit added its entry file
        - When changelog-updated checks a commit that adds another entry file
            - Then it fails naming the branch's entry file and the new one
- 0019.15
    - Given Developer has staged a new or edited `CHANGELOG.md` section
        - When changelog-updated checks the staged commit
            - Then it fails pointing me to an entry file in `changelog/`

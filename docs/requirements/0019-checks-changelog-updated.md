---
relatedConfigurations: ['../../.fitnessrc.json']
---

# 0019 Checks Changelog Updated

## Why

My branch carries one dated changelog entry, in its own entry file. Each commit adds bullets to it that name what changed.

## Measurement

staged commits whose changelog bullets name the change under the branch's one current entry
-
staged commits with changes

Source: `go/cmd/fitness-check-changelog-updated/main.go:88`

## Requirements

- 0019.3
    - Given Developer has an entry sharing fewer than three words with the change
        - When changelog-updated checks the staged commit
            - Then it fails with the count and suggested words
- 0019.6
    - Given Developer has a merge in progress that replays an old entry
        - When changelog-updated checks the staged commit
            - Then it passes without judging the old heading
- 0019.13
    - Given Developer has staged changes without a staged entry file, or no `CHANGELOG.md`
        - When changelog-updated checks the staged commit
            - Then it fails asking me to add an entry file
- 0019.14
    - Given Developer has a branch whose earlier commit added its entry file
        - When changelog-updated checks a commit that adds another entry file
            - Then it fails naming the branch's entry file and the new one
- 0019.15
    - Given Developer has staged a new or edited section in a committed `CHANGELOG.md`
        - When changelog-updated checks the staged commit
            - Then it fails pointing me to an entry file in `changelog/`
- 0019.16
    - Given Developer has staged a new entry file naming the change, and a first `CHANGELOG.md` or none
        - When changelog-updated checks the staged commit
            - Then it passes
- 0019.17
    - Given Developer has a branch whose earlier commit added the first `CHANGELOG.md`
        - When changelog-updated checks a commit that edits it
            - Then it passes
- 0019.18
    - Given Developer has staged a new entry file whose name lacks its heading's date
        - When changelog-updated checks the staged commit
            - Then it fails naming the dated file name it expects

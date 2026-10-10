---
relatedConfigurations: ['../../.fitnessrc.json']
---

# 0021 Checks Cspell

## Why

I catch typos before they ship, while my project's own terms still pass.

## Measurement

checked files with no unknown words
-
checked files

Source: `go/cmd/fitness-check-cspell/main.go:68`

## Requirements

- 0021.1
    - Given a tracked file with a misspelled word
        - When cspell runs
            - Then it fails naming the file, line, column, and word
- 0021.2
    - Given a repo `cspell.json` beside an installed shared one
        - When cspell runs
            - Then only the repo's words are accepted
- 0021.5
    - Given a `cspell.json` that sets `ignorePaths`
        - When cspell runs
            - Then it fails asking me to remove `ignorePaths`
- 0021.6
    - Given unknown words only in binary, `linguist-generated`, or untracked files
        - When cspell runs
            - Then it passes
- 0021.7
    - Given a staged misspelling, a staged deletion, and a committed misspelling
        - When cspell runs
            - Then it fails naming only the staged word
- 0021.8
    - Given an issue or PR body with a misspelling
        - When cspell checks it in body mode
            - Then it fails naming the word
- 0021.9
    - Given no repo `cspell.json`
        - When cspell runs
            - Then an installed `@mayjournal/fitness-shared` package's words are accepted, else the words built into fitness
- 0021.10
    - Given a `cspell.json` `flagWords` phrase used in a tracked file, in a body, and in that `cspell.json`
        - When cspell runs
            - Then it fails naming each use as a forbidden word, except in `cspell.json`

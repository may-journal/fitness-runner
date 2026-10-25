---
relatedConfigurations: ['../../.fitnessrc.json']
---

# 0021 Checks Cspell

## Why

I catch typos before they ship, while my project's own terms still pass.

A full cspell run takes 0.06 s on fitness-runner, down from 0.18 s, and 0.03 s on may-photos, down from 0.04 s. These are medians of 21 runs on an M5 Max, measured for #466.

## Measurement

checked files with no unknown words
-
checked files

Source: `go/cmd/fitness-check-cspell/main.go:53`

## Requirements

- 0021.13
    - Given Developer has tracked misspellings in a short file, and around a `cspell:disable` block crossing 64 KB in a long one
        - When cspell runs
            - Then it fails naming each word's file, line and column in order, except inside the block
- 0021.2
    - Given Developer has a repo `cspell.json` beside an installed shared one
        - When cspell runs
            - Then only the repo's words are accepted
- 0021.5
    - Given Developer has a `cspell.json` that sets `ignorePaths`
        - When cspell runs
            - Then it fails asking me to remove `ignorePaths`
- 0021.11
    - Given Developer has unknown words only in binary, `linguist-generated`, lockfile, build wrapper, or untracked files
        - When cspell runs
            - Then it passes
- 0021.7
    - Given Developer has a staged misspelling, a staged deletion, and a committed misspelling
        - When cspell runs
            - Then it fails naming only the staged word
- 0021.8
    - Given Developer has an issue or PR body with a misspelling
        - When cspell checks it in body mode
            - Then it fails naming the word
- 0021.12
    - Given Developer has no repo `cspell.json`
        - When cspell runs
            - Then an installed `@mayjournal/fitness-shared` package's words are accepted, else the words built into fitness for every language
- 0021.10
    - Given Developer has a `cspell.json` `flagWords` phrase used in a tracked file, in a body, and in that `cspell.json`
        - When cspell runs
            - Then it fails naming each use as a forbidden word, except in `cspell.json`

---
relatedConfigurations: ['../../.fitnessrc.json']
---

# 0057 Checks Swiftlint

## Why

I keep Swift code clean, because every SwiftLint finding, even a warning, turns the run red where I can see it.

## Measurement

Swift repos with a clean strict SwiftLint run reported green
-
Swift repos checked

Source: `go/cmd/fitness-check-swiftlint/main.go:99`

## Requirements

- 0057.1
    - Given a Swift file with a warning-level finding
        - When fitness runs swiftlint
            - Then it fails naming the file, line, column, reason, and rule
- 0057.2
    - Given a Swift file with a whole-file finding
        - When fitness runs swiftlint
            - Then it fails naming the file and line without a column
- 0057.3
    - Given clean Swift files
        - When fitness runs swiftlint
            - Then it passes
- 0057.4
    - Given a repo `.swiftlint.yml` that disables a found rule
        - When fitness runs swiftlint
            - Then it passes, since the repo config wins
- 0057.5
    - Given a repo with no Swift files
        - When fitness runs swiftlint
            - Then it passes with zero files
- 0057.6
    - Given Swift files and no SwiftLint on the path
        - When fitness runs swiftlint
            - Then it fails with the install hint
- 0057.7
    - Given a `.swiftlint.yml` that SwiftLint cannot read
        - When fitness runs swiftlint
            - Then it fails with the command to run it myself
- 0057.8
    - Given a `.swiftlint.yml` that excludes every Swift file
        - When fitness runs swiftlint
            - Then it passes, since nothing is left to lint

# 0057 Checks Swiftlint

## Why

I keep Swift code clean, because every SwiftLint finding, even a warning, turns the run red where I can see it.

## Measurement

Swift repos with a clean strict SwiftLint run reported green
-
Swift repos checked

## Requirements

- 0057.1
    - Given Developer has a Swift file with a warning-level finding
        - When fitness runs swiftlint
            - Then it fails naming the file, line, column, reason, and rule
- 0057.2
    - Given Developer has a Swift file with a whole-file finding
        - When fitness runs swiftlint
            - Then it fails naming the file and line without a column
- 0057.3
    - Given Developer has clean Swift files
        - When fitness runs swiftlint
            - Then it passes
- 0057.4
    - Given Developer has a repo `.swiftlint.yml` that disables a found rule
        - When fitness runs swiftlint
            - Then it passes, since the repo config wins
- 0057.5
    - Given Developer has a repo with no Swift files
        - When fitness runs swiftlint
            - Then it passes with zero files
- 0057.6
    - Given Developer has Swift files and no SwiftLint on the path
        - When fitness runs swiftlint
            - Then it fails with the install hint
- 0057.8
    - Given Developer has a `.swiftlint.yml` that excludes every Swift file
        - When fitness runs swiftlint
            - Then it passes, since nothing is left to lint
- 0057.9
    - Given Developer has a pull request that changes one of two Swift files
        - When fitness runs swiftlint on the pull request
            - Then it lints only that file, or the whole tree when the config lists `included:` paths

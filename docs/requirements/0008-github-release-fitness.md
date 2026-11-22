# 0008 Release Fitness

## Why

I release fitness and fitness-ci from `version.txt` and checked archives, with notes that list each new entry and major tags that move.

## Measurement

releases whose tag, archives and notes match the repo

-

releases

## Requirements

- 0008.1
  - Given Developer has a `version.txt`
    - When the release tool runs
      - Then it prints and verifies that version, or names the mistake
- 0008.2
  - Given Developer has a release step
    - When it passes or fails
      - Then a GitHub job summary shows its outcome, and only inside Actions
- 0008.3
  - Given Developer has built release archives
    - When `bundle-hashes`, `assemble` or `record` runs
      - Then each platform's hash and size print, and a changed file fails
- 0008.4
  - Given Developer has changelog entry files
    - When a release writes its notes
      - Then they list each new entry, newest first, with a working compare link
- 0008.5
  - Given Developer has a release pull request
    - When Release Please runs
      - Then it links the tracking issue once and skips the steps a release lacks
- 0008.6
  - Given Developer has a created release
    - When Release Please moves the major tags
      - Then each tag points at its own release, or nothing moves
- 0008.7
  - Given Developer has the workflows and actions of this repo
    - When each step runs its fitness-ci subcommand the way its workflow does
      - Then the subcommand exists and does the step's job
- 0008.8
  - Given Developer has a subcommand that a reusable workflow runs through the fitness-ci action
    - When the subcommand writes a step output
      - Then the action declares that output, so the caller reads it

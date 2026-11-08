# 0148 Checks Large Diffs

## Why

When my commit touches hundreds of files, the requirement and changelog checks still finish in seconds. A large change never spends the fast total on them.

## Measurement

large changes whose requirement and changelog checks each pass within five seconds

-

large changes

## Requirements

- 0148.1
  - Given Developer has 266 changed files with their test, requirement doc and changelog entry
    - When the requirement checks judge them staged or in a pull request, and changelog-updated judges them staged
      - Then each passes within five seconds

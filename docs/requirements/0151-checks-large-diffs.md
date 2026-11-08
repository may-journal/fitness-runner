# 0151 Checks Large Diffs

## Why

When my commit touches hundreds of files, the requirement and changelog checks still finish in seconds. A large change never spends the fast total on them.

## Measurement

large changes whose requirement and changelog checks pass with a few git calls in all

-

large changes

## Requirements

- 0151.1
  - Given Developer has 266 changed files with their test, requirement doc and changelog entry
    - When the requirement checks judge them staged or in a pull request, and changelog-updated judges them staged
      - Then each passes, and the whole run starts git at most 40 times

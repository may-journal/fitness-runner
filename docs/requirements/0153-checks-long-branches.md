# 0153 Checks Long Branches

## Why

My branch can hold a thousand commits, and the commit checks still read its history in one go. A long branch never spends the fast total on them.

## Measurement

long branches whose commit checks each pass within five seconds

-

long branches

## Requirements

- 0153.1
  - Given Developer has a branch of 1,000 commits and its open PR
    - When commit-attribution, semantic-commit and local-branch check it
      - Then each passes within five seconds

# 0148 Checks Large Diffs

## Why

When my commit touches hundreds of files, the requirement and changelog checks still finish in seconds. A large change never spends the fast total on them.

## Measurement

large staged commits whose requirement and changelog checks each pass within five seconds

-

large staged commits

## Requirements

- 0148.1
  - Given Developer has staged 266 changed files with their test, requirement doc and changelog entry
    - When requirements-changed, requirements and changelog-updated check the staged commit
      - Then each passes within five seconds

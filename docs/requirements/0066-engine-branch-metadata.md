---
relatedConfigurations: ['../../.fitnessrc.json']
---

# 0066 Branch Metadata

## Why

My branch shows PR and issue findings before I wait for Actions.

## Measurement

branch runs with current metadata feedback
-
branch runs

Source: `go/internal/metadata/branch.go:20`

## Requirements

- 0066.1
    - Given Developer has file violations and a PR with a bad title
        - When I run the full local suite
            - Then both findings appear and the run fails
- 0066.2
    - Given Developer has no open PR on the branch
        - When I run branch-check
            - Then it reports a skip
- 0066.3
    - Given Developer has a detached HEAD
        - When I run branch-check
            - Then it skips without GitHub reads
- 0066.4
    - Given Developer has two open PRs matching the branch
        - When I run branch-check
            - Then it fails naming the ambiguity
- 0066.5
    - Given Developer has credentials GitHub rejects
        - When I run branch-check
            - Then it fails with the cause and retry command
- 0066.6
    - Given Developer has a PR with no closing links
        - When I run branch-check
            - Then it fails the shared closing-link rule
- 0066.7
    - Given Developer has a fork whose stale main lacks an upstream Plan commit and a PR closing an ordinary issue
        - When I run branch-check
            - Then it checks my branch Plan and ordinary issue without reading the upstream Plan
- 0066.8
    - Given Developer has a branch with a GitHub remote
        - When I run one file check
            - Then it does not read branch metadata

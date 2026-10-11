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
    - Given file violations and a PR with a bad title
        - When I run the full local suite
            - Then both findings appear and the run fails
- 0066.2
    - Given no open PR on my branch
        - When I run branch-check
            - Then it reports a skip
- 0066.3
    - Given a detached HEAD
        - When I run branch-check
            - Then it skips without GitHub reads
- 0066.4
    - Given two open PRs match my branch
        - When I run branch-check
            - Then it fails naming the ambiguity
- 0066.5
    - Given GitHub rejects my credentials
        - When I run branch-check
            - Then it fails with the cause and retry command
- 0066.6
    - Given a PR with no closing links
        - When I run branch-check
            - Then it fails the shared closing-link rule
- 0066.7
    - Given my fork's stale main lacks an upstream Plan commit and my PR closes an ordinary issue
        - When I run branch-check
            - Then it checks my branch Plan and ordinary issue without reading the upstream Plan
- 0066.8
    - Given a branch with a GitHub remote
        - When I run one file check
            - Then it does not read branch metadata

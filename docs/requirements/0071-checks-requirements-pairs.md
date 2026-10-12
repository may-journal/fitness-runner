---
relatedConfigurations: ['../../.fitnessrc.json']
---

# 0071 Checks Requirements Pairs

## Why

A requirement's test and its doc change together, so neither drifts from the other.

## Measurement

changed requirement tests and docs whose partner changed too
-
changed requirement tests and docs

Source: `go/cmd/fitness-check-requirements/pairs.go:99`

## Requirements

- 0071.1
    - Given a changed requirement test and its unchanged doc
        - When I commit it with the hooks
            - Then the commit is refused naming both files and the missing doc
- 0071.2
    - Given a changed requirement doc and its unchanged test
        - When fitness runs on pull request changes
            - Then it fails naming both files and the missing test
- 0071.3
    - Given a changed test file named for two requirements
        - When fitness runs on staged changes
            - Then it fails naming both docs until either doc changes
- 0071.4
    - Given a changed shared test helper that proves no requirement
        - When fitness runs on staged changes
            - Then the check passes
- 0071.5
    - Given an unpaired change and a commit trailer giving the reason
        - When I commit it with the hooks
            - Then the commit goes through
- 0071.6
    - Given a pull request commit with that trailer
        - When fitness runs on pull request changes
            - Then it fails until the pull request body lists the trailer
- 0071.7
    - Given a merge, cherry-pick, or revert that replays an unpaired change
        - When I commit or amend it with the hooks
            - Then the commit goes through
- 0071.8
    - Given a branch that merged an unpaired change from its moved base
        - When fitness runs on pull request changes
            - Then only the branch's own changes are paired

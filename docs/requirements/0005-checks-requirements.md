---
relatedConfigurations: ['../../.fitnessrc.json']
---

# 0005 Checks Requirements

## Why

Every requirement keeps its IDs and changes with its test, so a doc and its proof never drift apart.

## Measurement

changed requirement tests and docs whose partner changed too
-
changed requirement tests and docs

Source: `go/cmd/fitness-check-requirements/pairs.go:99`

## Requirements

- 0005.8
    - Given Developer has an ID that an earlier commit deleted
        - When a doc brings it back
            - Then it fails
- 0005.9
    - Given Developer has a requirement test or doc changed without its partner
        - When fitness judges my commit or pull request
            - Then it fails naming both files and the missing half
- 0005.10
    - Given Developer has a changed test file named for two requirements
        - When fitness runs on staged changes
            - Then it fails naming both docs until either doc changes
- 0005.11
    - Given Developer has a changed shared test helper that proves no requirement
        - When fitness runs on staged changes
            - Then the check passes
- 0005.12
    - Given Developer has an unpaired change and a commit trailer giving the reason
        - When I commit it with the hooks
            - Then the commit goes through
- 0005.13
    - Given Developer has a pull request commit with that trailer
        - When fitness runs on pull request changes
            - Then it fails until the pull request body lists the trailer
- 0005.14
    - Given Developer has a merge, cherry-pick, revert, or base merge that replays an unpaired change
        - When fitness judges my commit or pull request
            - Then only the branch's own changes are judged

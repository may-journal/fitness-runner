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

Source: `go/cmd/fitness-check-requirements/pairs.go:88`

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
- 0005.14
  - Given Developer has a merge, cherry-pick, revert, or base merge that replays an unpaired change
    - When fitness judges my commit or pull request
      - Then only the branch's own changes are judged
- 0005.15
  - Given Developer has a commit with a requirement test alone and its doc staged
    - When I amend the commit with the hooks
      - Then the commit goes through, judged on everything since its parent
- 0005.16
  - Given Developer has a requirement test that fails the check at its base
    - When I fix the test without changing its doc
      - Then my commit, amend, or pull request passes, while a test that passed at its base still fails without its doc
- 0005.17
  - Given Developer has a requirement doc changed only in its formatting, such as blank lines, line breaks, indents or front matter
    - When fitness judges my commit or pull request
      - Then it passes without a test change

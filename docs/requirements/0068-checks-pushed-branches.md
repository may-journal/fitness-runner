---
relatedConfigurations: ['../../.fitnessrc.json']
---

# 0068 Pushed Branches

## Why

Every branch I push gets its own metadata checked, regardless of which branch is checked out.

## Measurement

pushed branches validated against their destination name and local tip
-
live branch updates pushed

## Requirements

- 0068.1
    - Given Developer has an invalid PR on a branch that is not checked out
        - When I push that branch
            - Then the push fails with that PR's finding
- 0068.2
    - Given Developer has a valid pushed branch and an invalid checked-out branch
        - When I push only the valid branch
            - Then the push succeeds without checking the checkout's PR
- 0068.3
    - Given Developer has two pushed branches and one invalid PR
        - When I push both branches together
            - Then one meta-check validates both branches once and refuses the push
- 0068.4
    - Given Developer has an existing remote branch
        - When I delete it
            - Then the push succeeds without reading branch metadata
- 0068.5
    - Given Developer has a renamed push whose local tip names a malformed Plan
        - When I push from another checkout branch
            - Then it finds the destination PR and fails the pushed tip's Plan
- 0068.6
    - Given Developer has two pushed branches whose PRs close one approved Plan
        - When I push both together
            - Then GitHub is asked about that Plan once

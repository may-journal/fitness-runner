---
relatedConfigurations: ['../../.fitnessrc.json']
---

# 0091 Auto-merge

## Why

My stacked pull requests keep a green Auto-merge check, and each one gets auto-merge once it targets the default branch.

## Measurement

Auto-merge runs on stacked pull requests that pass
-
Auto-merge runs on stacked pull requests

Source: #506

## Requirements

- 0091.1
    - Given Developer has a pull request stacked on another pull request's branch
        - When Auto-merge runs on it
            - Then it passes with a notice naming both branches and asks GitHub for nothing
- 0091.2
    - Given Developer has a stacked pull request that GitHub moved onto the default branch after its base merged
        - When I push to it
            - Then Auto-merge turns on squash auto-merge

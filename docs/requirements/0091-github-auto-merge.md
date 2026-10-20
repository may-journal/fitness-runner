---
relatedConfigurations: ['../../.fitnessrc.json']
---

# 0091 Auto-merge

## Why

Every ready pull request of mine ends up with auto-merge on, or says plainly in a comment why it has none. A stacked one keeps a green Auto-merge check, and its comment tells me how to merge it. Any other auto-merge failure turns the check red with GitHub's answer.

## Measurement

Ready pull requests with auto-merge on or an auto-merge comment
-
Ready pull requests

Source: #539

## Requirements

- 0091.1
    - Given Developer has a pull request stacked on another pull request's branch
        - When Auto-merge runs on it twice
            - Then it passes, asks GitHub for nothing, and comments once naming both branches and how to merge it
- 0091.2
    - Given Developer has a stacked pull request that GitHub moved onto the default branch after its base merged
        - When Auto-merge runs on it
            - Then it turns on squash auto-merge and edits its comment to say so; a pull request without that comment gets none
- 0091.3
    - Given Developer has a pull request on the default branch that GitHub still counts as stacked
        - When Auto-merge runs and GitHub refuses auto-merge for stacked pull requests
            - Then it passes, shows GitHub's answer, and comments how to merge it
- 0091.4
    - Given Developer has a pull request on the default branch
        - When Auto-merge runs and GitHub refuses auto-merge for any other reason
            - Then it fails with an error naming GitHub's answer and leaves no comment
- 0091.5
    - Given Developer has a ready pull request
        - When I reopen it, or push to it while auto-merge is on
            - Then Auto-merge runs on the reopen, and skips the push before a runner starts

---
relatedConfigurations: ['../../.fitnessrc.json']
---

# 0101 Commit and branch skills

## Why

When a commit, branch or workflow fails its check, I follow one skill and fix it on the first try.

## Measurement

Commit and branch skills whose quoted failure matches the real check
-
Commit and branch skills

Source: #531

## Requirements

- 0101.1
    - Given Developer has a commit message with a misspelled Plan line, and one spelled `Plan #63`
        - When I run plan-trailer on each as its skill says
            - Then the first fails with the line the skill quotes and the second passes
- 0101.2
    - Given Developer has a repo using AI trailers and a message that lacks them, and one that has them
        - When I run commit-attribution on each as its skill says
            - Then the first fails with the lines the skill quotes and the second passes
- 0101.3
    - Given Developer has a commit subject without a scope, and one with a scope
        - When I run semantic-commit on each as its skill says
            - Then the first fails with the line the skill quotes and the second passes
- 0101.4
    - Given Developer has a branch that links an issue in an earlier commit
        - When I run issue-link-once on a message that links it again, and on one that does not
            - Then the first fails with the line the skill quotes and the second passes
- 0101.5
    - Given Developer has a branch whose open PR closes no issue, and one whose PR closes one
        - When I run branch-check on each as its skill says
            - Then the first fails with the line the skill quotes and the second passes
- 0101.6
    - Given Developer has a workflow calling a may-journal workflow at a tag that lacks it, and one at a tag that has it
        - When I run workflow-ref-exists on each as its skill says
            - Then the first fails with the line the skill quotes and the second passes
- 0101.7
    - Given Developer has a repo to work in
        - When I run read-repo-first as its skill says
            - Then it passes and prints the box the skill describes

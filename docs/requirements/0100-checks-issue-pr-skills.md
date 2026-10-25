---
relatedConfigurations: ['../../.fitnessrc.json']
---

# 0100 Issue and PR skills

## Why

When a Plan, Epic or pull request description fails its check, I follow the skill beside that check's code. I fix it on the first try.

## Measurement

Issue and PR body skills whose quoted failure matches the real check
-
Issue and PR body skills

Source: #531

## Requirements

- 0100.1
    - Given Developer has a Plan body missing its pitch, and one that follows the template
        - When I run plan-structure on each as its skill says
            - Then the first fails with the line the skill quotes and the second passes
- 0100.2
    - Given Developer has an Epic list with a checkbox, an emoji and an item out of run order, and a clean list
        - When I run epic-structure on each as its skill says
            - Then the first fails with the lines the skill quotes and the second passes
- 0100.3
    - Given Developer has a PR description with no Background, and one that follows the template
        - When I run pr-structure on each as its skill says
            - Then the first fails with the line the skill quotes and the second passes
- 0100.4
    - Given Developer has a PR description that only mentions an issue, and one that closes it
        - When I run pr-closes-issue on each as its skill says
            - Then the first fails with the line the skill quotes and the second passes
- 0100.5
    - Given Developer has an issue with an unchecked box, and one with every box ticked
        - When I run issue-checklist on each as its skill says
            - Then the first fails with the line the skill quotes and the second passes

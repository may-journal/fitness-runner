---
relatedConfigurations: ['../../.fitnessrc.json']
---

# 0082 Issue Kinds

## Why

Every labeled issue, whatever its kind, shows one current verdict against its own template, so I add a kind with one template file.

## Measurement

labeled issue edits whose verdict matches their kind's template and rules
-
labeled issue edits

## Requirements

- 0082.1
    - Given Developer has an Epic that lists each of its sub-issues as a plain reference in run order
        - When issue-check judges it
            - Then it comments that the Epic looks good and labels it fitness-valid
- 0082.2
    - Given Developer has an Epic list with a checkbox, an emoji, a repeated reference, a table, an out-of-order item, or an oversized group
        - When issue-check judges it
            - Then it comments each error, labels it fitness-invalid, and exits 1
- 0082.3
    - Given Developer has an Epic whose list and GitHub sub-issues differ
        - When issue-check judges it
            - Then it names each listed issue that is not a sub-issue, or cannot be read, and each sub-issue left off the list
- 0082.4
    - Given Developer has an issue of a kind that has only a template, or a Plan whose repo template adds a section
        - When issue-check judges it
            - Then it fails a body missing the template's sections and passes one that follows it
- 0082.6
    - Given Developer has an issue with two kind labels
        - When issue-check judges it
            - Then it fails asking to keep one kind label
- 0082.7
    - Given Developer has a Plan whose body plan-check already judged
        - When issue-check judges the same body
            - Then it adds no comment
- 0082.8
    - Given Developer has a manual issue-check workflow run
        - When issue-check runs
            - Then it judges every open issue with a kind label
- 0082.9
    - Given Developer has an issue with no kind label, one moved away, one with only its title edited, or a verdict label added
        - When issue-check judges each event
            - Then it notes the issue is not checked, writes nothing, and exits 0

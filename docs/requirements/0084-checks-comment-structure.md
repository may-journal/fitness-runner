---
relatedConfigurations: ['../../.fitnessrc.json']
---

# 0084 Checks Comment Structure

## Why

Every comment I read is short and plain, with no notes about its own edits and no agent noise.

## Measurement

comments within their budget and free of change-log notes, agent frames, and model IDs
-
comments

Source: `go/cmd/fitness-check-comment-structure/main.go:78`

## Requirements

- 0084.1
    - Given Developer has a comment over the word budget, with more words inside a table and code
        - When comment-structure checks it
            - Then it fails naming the prose word count only
- 0084.2
    - Given Developer has a repo config that raises the word budget
        - When comment-structure checks the same comment
            - Then it passes
- 0084.3
    - Given Developer has a comment with two plain tables and a callout table
        - When comment-structure checks it
            - Then it fails naming two tables
- 0084.4
    - Given Developer has a comment with an unnumbered diagram, and another with a legend after a numbered diagram
        - When comment-structure checks each
            - Then only the unnumbered diagram fails, naming its line
- 0084.5
    - Given Developer has a comment with a change-log note, and another that only says when something is updated
        - When comment-structure checks each
            - Then only the note fails, naming its line
- 0084.6
    - Given Developer has a comment with an agent frame and a model ID, and another with a model ID in a code span
        - When comment-structure checks each
            - Then only the first fails, naming each one and its line
- 0084.7
    - Given Developer has a Plan body
        - When I run fitness `--blob` on it
            - Then comment-structure does not apply
- 0084.8
    - Given Developer has no comment to check
        - When comment-structure runs
            - Then it passes having checked no files

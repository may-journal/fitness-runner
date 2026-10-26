---
relatedConfigurations: ['../../.fitnessrc.json']
---

# 0087 Duplicate Issues

## Why

A new issue that repeats an older one points to it, so people find the first thread.

## Measurement

new issues resembling an older one that carry one note naming it
-
new issues resembling an older one

## Requirements

- 0087.1
    - Given Developer has a new issue that repeats an older one and another unlike any
        - When dupe-check judges each opened issue
            - Then it posts one note naming the older issue on the first and nothing on the second
- 0087.2
    - Given Developer has an issue whose duplicate note dupe-check already posted
        - When dupe-check judges an edit to that issue
            - Then it edits that note in place and posts no second one
- 0087.3
    - Given Developer has an issue that repeats its parent, a sibling sub-issue, and an issue it names
        - When dupe-check judges the opened issue
            - Then it writes nothing
- 0087.4
    - Given Developer has a close tier on in `.fitnessrc.json`, and a new issue above the close score against an open one
        - When dupe-check judges it opened and close-check judges its close
            - Then it closes as a duplicate with a note, and close-check leaves it closed
- 0087.5
    - Given Developer has issues that repeat each other, and one closed before the window
        - When dupe-check runs from a manual sweep
            - Then it prints the score of each pair in the window and writes nothing

# 0015 Closed Issues

## Why

An issue closed with unfinished work comes back open, so nothing is quietly dropped.

## Measurement

issues closed with unchecked items that are reopened
-
issues closed as completed with unchecked items

## Requirements

- 0015.1
    - Given Developer has an issue closed as completed with an unchecked item
        - When close-check runs
            - Then it reopens the issue with a comment naming the item and the closer
- 0015.2
    - Given Developer has a close that close-check already handled, or one where only another author pasted its marker
        - When it runs again
            - Then it stays quiet on the first and reopens the second
- 0015.3
    - Given Developer has an issue closed as not planned with unchecked items
        - When close-check runs
            - Then the issue stays closed
- 0015.4
    - Given Developer has an issue with no parent, closed with every item ticked
        - When close-check runs
            - Then the issue stays closed

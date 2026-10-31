# 0134 Close Parents

## Why

A parent issue closes itself when its last child closes, so finished work never waits on someone to tick it off.

## Measurement

parents closed by their last child
-
parents whose children were all closed as completed

## Requirements

- 0134.1
    - Given Developer has a parent whose only sub-issue is the issue just closed as completed
        - When its last child closes
            - Then it closes the parent as completed
- 0134.2
    - Given Developer has a parent that lists the closed issue as its last reference in the list
        - When its last child closes
            - Then it closes the parent as completed
- 0134.3
    - Given Developer has a parent that lists its last child both as a sub-issue and in the list
        - When its last child closes
            - Then it closes the parent with one comment naming that child and its close reason
- 0134.4
    - Given Developer has a parent whose own parent has no other open child
        - When its last child closes
            - Then it closes the parent and then the grandparent

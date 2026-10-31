# 0135 Open Parents

## Why

A parent stays open while any of its work is unfinished or dropped, so closing a child never hides what is left.

## Measurement

parents left open by a child that closed
-
parents with a child that closed

## Requirements

- 0135.1
    - Given Developer has a parent with another child still open
        - When a child closes
            - Then the parent stays open
- 0135.2
    - Given Developer has an issue that names no children
        - When a child closes
            - Then the issue stays open
- 0135.3
    - Given Developer has a parent with a child closed as not planned
        - When a child closes
            - Then the parent stays open
- 0135.4
    - Given Developer has a parent with an unchecked box of its own
        - When a child closes
            - Then the parent stays open
- 0135.5
    - Given Developer has a closed parent and one of its children reopens
        - When another child closes
            - Then the parent stays closed

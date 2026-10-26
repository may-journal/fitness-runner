# 0011 Issue Link Once

## Why

Each issue's timeline shows one link per branch, and fixing up the linking commit still works.

## Measurement

amended commits that keep their link
-
amended commits that link an issue

## Requirements

- 0011.1
    - Given Developer has a branch commit that links an issue
        - When I commit again linking the same issue
            - Then the commit is refused naming that issue
- 0011.2
    - Given Developer has a branch commit that links an issue
        - When I amend it and keep the link
            - Then the commit goes through
- 0011.3
    - Given Developer has a branch stacked on a pushed epic branch whose commit links an issue
        - When I commit a link to the same issue
            - Then the commit goes through, and a second link on my branch is refused

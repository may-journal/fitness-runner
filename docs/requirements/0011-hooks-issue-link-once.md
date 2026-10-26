---
relatedConfigurations: ['../../.fitnessrc.json']
---

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
- 0011.4
    - Given Developer is behind a proxy that never answers, on a branch whose commit links an issue
        - When I commit linking that issue, then another issue
            - Then each ends in seconds with a note naming the host and proxy, refusing only the repeat

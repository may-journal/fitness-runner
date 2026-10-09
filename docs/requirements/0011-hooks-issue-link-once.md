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

Source: `go/cmd/fitness/hook.go:79`

## Requirements

- 0011.1
    - Given a branch commit that links an issue
        - When I commit again linking the same issue
            - Then the commit is refused naming that issue
- 0011.2
    - Given a branch commit that links an issue
        - When I amend it and keep the link
            - Then the commit goes through

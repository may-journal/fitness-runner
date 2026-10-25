---
relatedConfigurations: ['../../.fitnessrc.json']
---

# 0120 Review Comments

## Why

A review with many comments costs one Actions job, and each failing comment still gets its own reply.

## Measurement

submitted reviews whose comments one run judges
-
submitted reviews

Source: `go/internal/metadata/comments.go:57`

## Requirements

- 0120.1
    - Given Developer has a submitted review with ten comments, two of them failing
        - When comment-check judges the review event
            - Then one run judges all ten and replies in the thread of each failing one

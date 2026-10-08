---
relatedConfigurations: ['../../.fitnessrc.json']
---

# 0003 Checks Self-Identify Availability

## Why

Every check decides for itself whether it can judge a repo, so any repo runs every check with nothing to configure.

## Measurement

checks that declare when they apply
-
checks in the catalog

Source: #189

## Requirements

- 0003.1
    - Given a repo a check cannot judge
        - When the check runs
            - Then it passes with zero files

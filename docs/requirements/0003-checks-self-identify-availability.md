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
    - Given Developer has a repo with no files a check can judge
        - When the check runs
            - Then it scans nothing and passes with zero files
- 0003.2
    - Given Developer has a repo that chooses no checks
        - When fitness runs
            - Then the result rows list every installed check, repo-wide checks first, then each language's

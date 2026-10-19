---
relatedConfigurations: ['../../.fitnessrc.json']
---

# 0092 Release Please

## Why

Every repo that opts in to releases calls one shared Release Please workflow. Its release pull request passes pr-check, and each release moves the major tag.

## Measurement

Release Please runs whose pull request passes pr-check
-
Release Please runs

Source: #517

## Requirements

- 0092.1
    - Given Developer has a first-release pull request body with a `## 1.0.0` heading and bold scopes
        - When Release Please shapes the notes, then reruns pr-check
            - Then the heading is `###`, scopes are plain, and the pull request closes and reopens
- 0092.2
    - Given Developer has a release pull request with no tracking issue
        - When Release Please links it twice
            - Then the first run opens the issue and the body closes it; the second changes nothing
- 0092.3
    - Given Developer has a release created with no pull request
        - When Release Please runs
            - Then only the major tag step runs, and each skipped step still reads its env

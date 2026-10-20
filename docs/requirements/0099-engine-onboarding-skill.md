---
relatedConfigurations: ['../../.fitnessrc.json']
---

# 0099 Onboarding skill

## Why

A new repo or agent adopts Fitness by following one skill, and every doc it names exists. I never ask anyone how.

## Measurement

Onboarding skills whose every link reaches a real doc
-
Onboarding skills

Source: #528

## Requirements

- 0099.1
    - Given Developer has the onboarding skill linked into an agent's skills folder
        - When I read it
            - Then it covers install, hooks, running checks, reading failures, and the Plan and pull request flow with the Plan trailer check linked
- 0099.2
    - Given Developer has the onboarding skill linked into an agent's skills folder
        - When I follow each doc link in it
            - Then every link names a file and heading that exist in the repo

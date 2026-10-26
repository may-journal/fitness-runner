---
relatedConfigurations: ['../../.fitnessrc.json']
---

# 0121 Actions Minutes

## Why

I see which workflows start the most jobs and bill the most minutes, so I measure a CI change by its bill.

## Measurement

workflows whose billed minutes match GitHub's whole-minute rounding of each job
-
workflows totalled

## Requirements

- 0121.1
    - Given Developer has an org whose private repo ran a 61 s job and two shorter jobs
        - When I run fitness-release minutes for that org over four days
            - Then it prints each workflow's jobs, billed minutes and median job time, and the totals
- 0121.2
    - Given Developer has a run whose only job was skipped
        - When I run fitness-release minutes for that repo
            - Then the skipped job counts no job and no minute

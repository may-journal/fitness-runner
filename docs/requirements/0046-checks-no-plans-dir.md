---
relatedConfigurations: ['../../.fitnessrc.json']
---

# 0046 Checks No Plans Dir

## Why

I keep every plan as a `Plan` issue, so a stray plan file never lands in the repo unnoticed.

## Measurement

files under `docs/plans/` reported red
-
files under `docs/plans/`

## Requirements

- 0046.1
    - Given Developer has a plan file in `docs/plans/`
        - When fitness runs no-plans-dir
            - Then it fails naming the file and telling me to open a Plan Issue
- 0046.2
    - Given Developer has a plan file in a folder below `docs/plans/`
        - When fitness runs no-plans-dir
            - Then it fails naming that nested file too
- 0046.3
    - Given Developer has a repo with no `docs/plans/` folder
        - When fitness runs no-plans-dir
            - Then it passes with no files judged

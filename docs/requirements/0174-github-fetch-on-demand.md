# 0174 GitHub fetch on demand

## Why

The issue, comment and PR bots installed every pinned check on each event, though one body needs only a few. Their cache keyed on `.fitnessrc.json`, which says nothing about the check versions a release holds.

## Measurement

Workflow steps that install checks the way their run needs
-
Workflow steps that install checks

## Requirements

- 0174.1
    - Given Developer has the fitness-install action
        - When I run its index key script by its path and read its restore step
            - Then each key holds the release index hash, not `.fitnessrc.json`, and restore falls back to any older set
- 0174.2
    - Given Developer has the fitness-command action and this repo's CI workflow
        - When I read how each installs the checks
            - Then events install on demand, while a manual run and a push to `main` install every check
- 0174.3
    - Given Developer has a checks cache holding a program the run linked and one it did not
        - When I run the save-checks key script by its path after a restore from an older index, then from this index
            - Then the first drops the unlinked program, the second keeps both, and each key starts with the job's prefix

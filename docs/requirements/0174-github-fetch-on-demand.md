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
        - When I read its checks cache step
            - Then its key holds the release index hash, not `.fitnessrc.json`, and a restore key falls back to older sets
- 0174.2
    - Given Developer has the fitness-command action and this repo's CI workflow
        - When I read how each installs the checks
            - Then events install on demand, while a manual run and a push to `main` install every check

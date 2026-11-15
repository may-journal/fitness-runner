# 0168 Engine fetch checks

## Why

The engine holds no checks. So a `fitness` from `go install` fetches the pinned checks on first run, and such a `fitness-install` installs a current release.

## Measurement

Runs on a clean machine that judge with the pinned checks
-
Runs on a clean machine

## Requirements

- 0168.1
    - Given Developer has a repo that pins one check and breaks it, and a home with no Fitness cache
        - When I run `fitness` on its own
            - Then it installs the pinned check, verified, into the user's cache and fails naming the break
- 0168.2
    - Given Developer has `fitness-install` built from source without release stamps
        - When I read its help
            - Then its default version is latest, never a stale stamped one

---
relatedConfigurations: ['../../.fitnessrc.json']
---

# 0008 Runner Says What Went Wrong

## Why

When I ask fitness for something it cannot do, it tells me what to fix instead of guessing.

## Measurement

mistakes fitness names
-
mistakes made

Source: `go/cmd/fitness/main.go:205`

## Requirements

- 0008.1
    - Given Developer has a check name that does not exist
        - When fitness runs it
            - Then it reports red with Unknown check and exits 1
- 0008.2
    - Given Developer has a .fitnessrc.json that is not valid JSON
        - When fitness runs
            - Then it reports red naming the file and exits 1
- 0008.3
    - Given Developer asks for help
        - When fitness runs with --help
            - Then it shows how to use it and exits 0
- 0008.4
    - Given Developer has a hook name fitness does not know
        - When fitness hook runs it
            - Then it reports red naming the unknown hook and exits 1
- 0008.5
    - Given Developer has a folder that is not a git repo
        - When I run fitness init
            - Then it says why, writes no hooks, and exits 1
- 0008.6
    - Given Developer has a folder that is not a git repo
        - When a fitness hook runs there
            - Then it says why and exits 1
- 0008.7
    - Given Developer has the commit-msg hook with no message file
        - When it runs
            - Then it reports the missing file and exits 1
- 0008.8
    - Given Developer has a fitness runner alone in its folder, with no checks in that folder
        - When fitness runs
            - Then it says to reinstall fitness and exits 1

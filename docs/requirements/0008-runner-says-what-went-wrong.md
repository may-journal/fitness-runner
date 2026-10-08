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
    - Given a check name that does not exist
        - When fitness runs it
            - Then it reports red with Unknown check and exits 1
- 0008.2
    - Given a .fitnessrc.json that is not valid JSON
        - When fitness runs
            - Then it reports red naming the file and exits 1
- 0008.3
    - Given a user who asks for help
        - When fitness runs with --help
            - Then it shows how to use it and exits 0

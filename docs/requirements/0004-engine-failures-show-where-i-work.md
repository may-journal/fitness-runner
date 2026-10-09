---
relatedConfigurations: ['../../.fitnessrc.json']
---

# 0004 Failures Show Where I Work

## Why

I see what broke on the line that broke it, without digging through CI logs.

## Measurement

findings annotated on their source line
-
findings that name a source line

Source: `go/cmd/fitness/ci.go:139`

## Requirements

- 0004.1
    - Given a finding that names a file and line
        - When fitness reports it in GitHub Actions
            - Then the annotation points at that file and line
- 0004.2
    - Given more findings than the annotation cap
        - When fitness reports them
            - Then a warning says how many were left out
- 0004.3
    - Given a run where every check passed
        - When fitness reports it in GitHub Actions
            - Then it adds no annotations
- 0004.4
    - Given a finding with a percent sign or line break
        - When fitness annotates it
            - Then it arrives intact as one annotation
- 0004.5
    - Given a run with failing checks
        - When fitness writes the job summary
            - Then it shows every check and each failure in full
- 0004.6
    - Given a run that fails before any check starts
        - When fitness reports it in GitHub Actions
            - Then one annotation names the setup failure without a file
- 0004.7
    - Given a job summary that earlier steps filled near its limit
        - When fitness writes the job summary
            - Then the summary says it was cut and where the complete report is

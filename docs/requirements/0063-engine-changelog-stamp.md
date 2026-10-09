---
relatedConfigurations: ['../../.fitnessrc.json']
---

# 0063 Changelog Stamp

## Why

I write a changelog entry under any placeholder heading, and my commit stamps it with the real time.

## Measurement

commits whose newest changelog heading shows the commit time
-
commits that stage CHANGELOG.md

Source: `go/cmd/fitness-stamp-changelog/main.go:50`

## Requirements

- 0063.1
    - Given the hooks and a staged CHANGELOG entry under an old heading
        - When I commit it
            - Then the committed heading and my file show the commit time
- 0063.2
    - Given the hooks and a staged CHANGELOG with several headings
        - When I commit it
            - Then only the newest heading is restamped
- 0063.3
    - Given the hooks and a CHANGELOG edit I did not stage
        - When I commit without staging it
            - Then the CHANGELOG keeps its heading and stays out of the commit

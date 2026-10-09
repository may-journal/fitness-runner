---
relatedConfigurations: ['../../.fitnessrc.json']
---

# 0032 Checks Jscpd

## Why

I fix logic in one place, because copied blocks of code or docs never pile up unseen.

## Measurement

repos with duplicated lines at or under 1% of scanned lines
-
repos scanned

Source: `go/cmd/fitness-check-jscpd/main.go:41`

## Requirements

- 0032.1
    - Given two files sharing a block past 1% of all lines
        - When `jscpd` runs
            - Then it fails naming the duplicate share and the 1% threshold
- 0032.2
    - Given two files sharing a block under 5 lines
        - When `jscpd` runs
            - Then it passes
- 0032.3
    - Given two files sharing a block under 50 tokens
        - When `jscpd` runs
            - Then it passes
- 0032.4
    - Given a Markdown doc and a JSON file sharing a block
        - When `jscpd` runs
            - Then it fails, since every tracked text file is compared
- 0032.5
    - Given copied lock files that `.gitattributes` marks `linguist-generated`
        - When `jscpd` runs
            - Then it passes, since generated files are left out
- 0032.6
    - Given binary files sharing a block
        - When `jscpd` runs
            - Then it passes, since binary files are skipped
- 0032.7
    - Given untracked files sharing a block
        - When `jscpd` runs
            - Then it passes, since only tracked files are scanned
- 0032.8
    - Given a gitignored file that is still tracked and shares a block
        - When `jscpd` runs
            - Then it fails, since tracked files are compared

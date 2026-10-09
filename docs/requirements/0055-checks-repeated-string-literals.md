---
relatedConfigurations: ['../../.fitnessrc.json']
---

# 0055 Checks Repeated String Literals

## Why

I keep each repeated value in one shared constant, so a rename touches one line instead of many.

## Measurement

repeated values reported red
-
values used three or more times in source

Source: `go/cmd/fitness-check-repeated-string-literals/main.go:183`

## Requirements

- 0055.1
    - Given a string used three times across source files
        - When fitness runs repeated-string-literals
            - Then it fails naming the value, its count, and each location
- 0055.2
    - Given a string used only twice
        - When fitness runs repeated-string-literals
            - Then it passes
- 0055.3
    - Given repeats only inside comments, regex, template strings, or module paths
        - When fitness runs repeated-string-literals
            - Then it passes, since none of those count
- 0055.4
    - Given repeated short values or idiomatic tokens like `'utf8'`
        - When fitness runs repeated-string-literals
            - Then it passes
- 0055.5
    - Given a string used seven times
        - When fitness runs repeated-string-literals
            - Then it lists five locations and says how many more
- 0055.6
    - Given two repeated strings with different counts
        - When fitness runs repeated-string-literals
            - Then it reports the most repeated first
- 0055.7
    - Given repeats spread across test and bench files
        - When fitness runs repeated-string-literals
            - Then it fails, since tests count as source
- 0055.8
    - Given a repeated value listed in `repeatedStringLiterals.allow`
        - When fitness runs repeated-string-literals
            - Then it passes

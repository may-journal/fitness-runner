---
relatedConfigurations: ['../../.fitnessrc.json']
---

# 0045 Checks No ESLint Disable

## Why

I fix the lint rule or the code instead of hiding a finding behind a disable comment.

## Measurement

source files with no ESLint disable directive
-
source files checked

Source: `go/cmd/fitness-check-no-eslint-disable/main.go:40`

## Requirements

- 0045.1
    - Given Developer has a source file with file, line, and next-line disable directives
        - When no-eslint-disable checks the repo
            - Then it fails listing each file, line, and directive
- 0045.2
    - Given Developer has source files with no disable directive
        - When no-eslint-disable checks the repo
            - Then it passes counting every source file
- 0045.3
    - Given Developer has directive text inside a string in every scanned extension
        - When no-eslint-disable checks the repo
            - Then it fails on each of those files
- 0045.4
    - Given Developer has directive text only in a Markdown file
        - When no-eslint-disable checks the repo
            - Then it passes, since Markdown is not scanned
- 0045.5
    - Given Developer has a repo with no JavaScript or TypeScript source
        - When no-eslint-disable checks the repo
            - Then it passes judging zero files
- 0045.6
    - Given Developer has committed directives under `node_modules` and `dist`
        - When no-eslint-disable checks the repo
            - Then it fails on both, since no folder is skipped
- 0045.7
    - Given Developer has a directive only in an untracked file
        - When no-eslint-disable checks the repo
            - Then it passes, since untracked files are not judged
- 0045.8
    - Given Developer has a committed directive and a clean staged file
        - When no-eslint-disable runs without `--all`
            - Then it passes judging only the staged file

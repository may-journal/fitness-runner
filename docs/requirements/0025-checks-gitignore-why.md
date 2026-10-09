---
relatedConfigurations: ['../../.fitnessrc.json']
---

# 0025 Checks Gitignore Why

## Why

I know why every file is ignored, because each `.gitignore` pattern carries its reason right above it.

## Measurement

`.gitignore` patterns with an explanatory comment directly above
-
`.gitignore` patterns

Source: `go/cmd/fitness-check-gitignore-why/main.go:88`

## Requirements

- 0025.1
    - Given a `.gitignore` where each pattern has a comment above it
        - When gitignore-why runs
            - Then it passes
- 0025.2
    - Given a pattern with no comment above it
        - When gitignore-why runs
            - Then it fails naming the line and the pattern
- 0025.3
    - Given one comment above two patterns
        - When gitignore-why runs
            - Then it fails naming the second pattern
- 0025.4
    - Given a blank line between a comment and its pattern
        - When gitignore-why runs
            - Then it fails naming the pattern
- 0025.5
    - Given a bare `#` with no text above a pattern
        - When gitignore-why runs
            - Then it fails naming the pattern
- 0025.6
    - Given a repo with no `.gitignore`
        - When gitignore-why runs
            - Then it passes with 0 files checked

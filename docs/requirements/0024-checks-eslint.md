---
relatedConfigurations: ['../../.fitnessrc.json']
---

# 0024 Checks Eslint

## Why

I keep every JavaScript and TypeScript file clean under one shared lint config, so code reads the same everywhere.

## Measurement

files ESLint reports clean
-
files linted

Source: `go/cmd/fitness-check-eslint/main.go:95`

## Requirements

- 0024.9
    - Given Developer has a repo without a `package.json`, or a scoped run that stages no JavaScript or TypeScript file
        - When fitness runs eslint
            - Then it passes with nothing to judge
- 0024.2
    - Given Developer has a JavaScript project without `eslint` installed
        - When fitness runs eslint
            - Then it fails with an install hint
- 0024.3
    - Given Developer has a project with the shared config and clean JavaScript and TypeScript files
        - When fitness runs eslint
            - Then it passes, counting every tracked file it linted
- 0024.4
    - Given Developer has a JavaScript file with object keys out of order
        - When fitness runs eslint
            - Then it fails with `path:line:col - message (sort-keys)`
- 0024.5
    - Given Developer has the repo's own config that only warns on `console` calls
        - When fitness runs eslint on a file calling `console.log`
            - Then it fails on that warning and ignores the shared rules
- 0024.6
    - Given Developer has the repo's own config that ignores a tracked file
        - When fitness runs eslint
            - Then it fails, asking to remove that ignore pattern
- 0024.7
    - Given Developer has a committed file with findings and a clean staged TypeScript file
        - When fitness runs eslint scoped to staged files
            - Then it passes after linting only the staged file
- 0024.11
    - Given Developer has a `biome.json`, `@biomejs/biome` 1.x or 2.x under any spec, no eslint, and more than 20 Biome findings
        - When fitness runs eslint
            - Then it lints with Biome instead, failing with `path:line:col - message (rule)` for every finding

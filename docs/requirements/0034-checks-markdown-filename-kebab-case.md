---
relatedConfigurations: ['../../.fitnessrc.json']
---

# 0034 Checks Markdown Filename Kebab Case

## Why

I find and link any doc in my kebab-case repo without guessing how its file name is spelled.

## Measurement

markdown files named in kebab-case or exempt
-
markdown files judged in kebab-case repos

Source: `go/cmd/fitness-check-markdown-filename-kebab-case/main.go:21`

## Requirements

- 0034.1
    - Given a kebab-case repo with a camelCase doc name
        - When markdown-filename-kebab-case runs
            - Then it fails saying that file must be kebab-case
- 0034.2
    - Given doc names in `snake_case`, `PascalCase`, or with spaces
        - When markdown-filename-kebab-case runs
            - Then it fails naming each of those files
- 0034.3
    - Given a repo whose doc names are all kebab-case, digits included
        - When markdown-filename-kebab-case runs
            - Then it passes
- 0034.4
    - Given capitalized docs such as `CODE_OF_CONDUCT.md` in a kebab-case repo
        - When markdown-filename-kebab-case runs
            - Then it passes, since standard docs are exempt
- 0034.5
    - Given a kebab-case doc inside a capitalized folder
        - When markdown-filename-kebab-case runs
            - Then it passes, since only the file name is judged
- 0034.6
    - Given a repo where most doc names are camelCase
        - When markdown-filename-kebab-case runs
            - Then it passes with 0 files, leaving the camelCase check to judge
- 0034.7
    - Given a repo with as many hyphenated as capitalized doc names
        - When markdown-filename-kebab-case runs
            - Then it fails naming the camelCase file, since kebab-case wins a tie
- 0034.8
    - Given a staged camelCase doc in a kebab-case repo with an older camelCase doc
        - When markdown-filename-kebab-case runs on staged files
            - Then it fails naming only the staged file

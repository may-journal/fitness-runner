---
relatedConfigurations: ['../../.fitnessrc.json']
---

# 0052 Checks Prettier

## Why

I want every tracked file in Prettier style, so diffs show real changes and never formatting noise.

## Measurement

tracked files Prettier reports as formatted
-
tracked files checked

Source: `go/cmd/fitness-check-prettier/main.go:339`

## Requirements

- 0052.1
    - Given a file my own Prettier config would reformat
        - When prettier checks the repo
            - Then it fails naming that file
- 0052.2
    - Given a formatted project with its config in `package.json`
        - When prettier checks the repo
            - Then it passes
- 0052.3
    - Given a repo with no `package.json`
        - When prettier checks the repo
            - Then it passes without needing Prettier
- 0052.4
    - Given a project with no config, formatted with single quotes
        - When prettier checks the repo
            - Then it passes under the shared config, not Prettier defaults
- 0052.5
    - Given a formatted file staged beside an unstaged messy one
        - When prettier checks the staged change
            - Then it passes, judging only the staged file
- 0052.6
    - Given a messy file and the `--write` argument
        - When prettier runs with that argument
            - Then it rewrites the file and passes
- 0052.7
    - Given a `.prettierignore` file, even without `package.json`
        - When prettier checks the repo
            - Then it fails telling me to delete it
- 0052.8
    - Given a project without Prettier installed
        - When prettier checks the repo
            - Then it fails with an install hint

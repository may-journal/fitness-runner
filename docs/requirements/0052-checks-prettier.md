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

Source: `go/cmd/fitness-check-prettier/main.go:363`

## Requirements

- 0052.9
    - Given Developer has the repo's own Prettier config, in a file or in `package.json`, in a CI job that colors output
        - When prettier checks the repo
            - Then it passes formatted files and fails naming each file that config would reformat
- 0052.3
    - Given Developer has a repo with no `package.json`
        - When prettier checks the repo
            - Then it passes without needing Prettier
- 0052.4
    - Given Developer has a project with no config, formatted with single quotes
        - When prettier checks the repo
            - Then it passes under the shared config, not Prettier defaults
- 0052.5
    - Given Developer has a formatted file staged beside an unstaged messy one
        - When prettier checks the staged change
            - Then it passes, judging only the staged file
- 0052.6
    - Given Developer has a messy file and the `--write` argument
        - When prettier runs with that argument
            - Then it hands that argument to Prettier in place of `--check`, and passes
- 0052.7
    - Given Developer has a `.prettierignore` file, even without `package.json`
        - When prettier checks the repo
            - Then it fails telling me to delete it
- 0052.8
    - Given Developer has a project without Prettier installed
        - When prettier checks the repo
            - Then it fails with an install hint
- 0052.11
    - Given Developer has a `biome.jsonc`, `@biomejs/biome` in `devDependencies`, no Prettier, and more than 20 files Biome would reformat
        - When prettier checks the repo
            - Then it checks formatting with Biome instead, failing naming every one of those files

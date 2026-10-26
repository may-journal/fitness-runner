# 0033 Checks Markdown Filename Camel Case

## Why

I find and link any doc in my camelCase repo without guessing how its file name is spelled.

## Measurement

markdown files named in camelCase or exempt
-
markdown files judged in camelCase repos

## Requirements

- 0033.1
    - Given Developer has a camelCase repo with a hyphenated doc name
        - When markdown-filename-camel-case runs
            - Then it fails saying that file must be camelCase
- 0033.2
    - Given Developer has a camelCase repo with `snake_case` and `PascalCase` doc names
        - When markdown-filename-camel-case runs
            - Then it fails naming each of those files
- 0033.3
    - Given Developer has a repo whose doc names are all camelCase
        - When markdown-filename-camel-case runs
            - Then it passes
- 0033.5
    - Given Developer has a camelCase doc inside a capitalized folder
        - When markdown-filename-camel-case runs
            - Then it passes, since only the file name is judged
- 0033.6
    - Given Developer has a repo where most doc names are hyphenated
        - When markdown-filename-camel-case runs
            - Then it passes with 0 files, leaving the kebab-case check to judge
- 0033.7
    - Given Developer has a repo with as many hyphenated as capitalized doc names
        - When markdown-filename-camel-case runs
            - Then it passes with 0 files, since kebab-case wins a tie
- 0033.8
    - Given Developer has a staged hyphenated doc in a camelCase repo with an older hyphenated doc
        - When markdown-filename-camel-case runs on staged files
            - Then it fails naming only the staged file
- 0033.9
    - Given Developer has capitalized docs such as `CODE_OF_CONDUCT.md` and GitHub's `.github/pull_request_template.md` in a camelCase repo
        - When markdown-filename-camel-case runs
            - Then it passes, since standard docs and names GitHub requires are exempt

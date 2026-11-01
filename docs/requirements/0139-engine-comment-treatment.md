# 0139 Comment Treatment

## Why

I know how each check treats my code comments before I write one. Every check declares whether it judges comments, strips them, or reads none. The docs and `fitness explain` show that one declaration, and real runs prove it.

## Measurement

Checks whose comment treatment a real run proves
-
Checks that read code

## Requirements

- 0139.1
    - Given AI agent wants to know how a check treats code comments
        - When it runs `fitness explain` for every check `fitness list` names
            - Then each states judged, stripped or not applicable, as `fitness list --json` does
- 0139.2
    - Given AI agent reads the check index in docs/checks.md
        - When it reads each check's comments field in `fitness list --json`
            - Then the index row's Comments column says the same
- 0139.3
    - Given Developer adds a check whose describe omits the comments field
        - When they run `fitness-release checks` to generate the docs
            - Then it fails naming the check and the allowed constants
- 0139.4
    - Given Developer misspells a word in a Go, Swift, TypeScript and shell comment
        - When cspell checks the repo
            - Then it fails naming each comment's path and line
- 0139.5
    - Given Developer adds the same misspelled word to a `cspell:ignore` comment in each file
        - When cspell checks the repo
            - Then it passes
- 0139.6
    - Given Developer repeats one string literal three times inside TypeScript comments
        - When repeated-string-literals checks the repo
            - Then it passes, while the same literal in code three times fails
- 0139.7
    - Given Developer writes a Go comment that looks like a function with many branches
        - When go-complexity checks the repo
            - Then it passes, while the same function in code fails
- 0139.8
    - Given Developer has two files that share only a long block of identical comments
        - When duplicate-code checks the repo
            - Then it passes, while the same block in code fails

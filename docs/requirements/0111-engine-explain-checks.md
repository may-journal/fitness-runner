# 0111 Explain Checks

## Why

I ask the installed binary why a check exists and what it judges, with no checkout and no network. The answer is the check's own code comments.

## Measurement

checks that `fitness explain` documents
-
checks in the catalog

## Requirements

- 0111.1
    - Given AI agent has a repo with fitness installed
        - When it runs `fitness explain` for every check `fitness list` names
            - Then each prints the check's name, scope, stage and documentation
- 0111.2
    - Given AI agent wants a check's documentation as data
        - When it runs `fitness explain local-branch --json`
            - Then it gets the name, summary, scope and body of that check, which the engine runs itself
- 0111.3
    - Given AI agent mistypes a check name
        - When it runs `fitness explain prose-budgets`
            - Then fitness exits 2 and suggests the closest check names
- 0111.4
    - Given AI agent has a repo with a local check whose describe handshake fails
        - When it runs `fitness explain` for that check
            - Then it prints a note saying why the check gave no metadata

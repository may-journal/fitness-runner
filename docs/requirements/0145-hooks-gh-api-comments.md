# 0145 Comments through gh api meet the hook

## Why

A comment Claude posts through `gh api`, a GraphQL mutation or a close comment meets the same hook as `gh pr comment`. A red comment never reaches GitHub, whichever way Claude sends it.

## Measurement

comments posted through gh api that the hook judges before they post
-
comments posted through gh api

## Requirements

- 0145.1
    - Given Developer has a fitness repo and Claude drafted a red comment
        - When Claude Code runs `fitness claude-hook` before a `gh api` call that posts or edits it on an issue or a PR line
            - Then it denies the call with the findings, whether a field or an @file gives the body
- 0145.2
    - Given Developer has a fitness repo and Claude drafted a review with a red line comment
        - When Claude Code runs `fitness claude-hook` before the `gh api` call, by fields or a `--input` JSON file
            - Then it denies the call, naming that line comment, and a clean review passes
- 0145.3
    - Given Developer has a fitness repo and Claude drafted a red comment
        - When Claude Code runs `fitness claude-hook` before a GraphQL mutation with it as the body variable, or a close with it
            - Then it denies the call with the findings
- 0145.4
    - Given Developer has a fitness repo
        - When Claude Code runs `fitness claude-hook` before a `gh api` call that reads, or posts a body the shell builds
            - Then it exits 0 with no output

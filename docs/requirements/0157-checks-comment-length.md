# 0157 Checks Comment Length

## Why

My functions show code instead of essays, because prose-budget fails a function body whose comments hold more words than its code can carry. Markdown files are judged as before.

## Measurement

Function bodies with comments within three words per code token

-

Function bodies with two or more comments

## Requirements

- 0157.1
    - Given Developer has Go, Swift and TypeScript functions whose two comments hold over three words per code token
        - When prose-budget checks the repo
            - Then it fails each body as `overlong` and says to keep the comments that say why
- 0157.2
    - Given Developer has functions with one long comment, brief comments or a fenced example, and an over-commented Markdown example
        - When prose-budget checks the repo
            - Then it passes
- 0157.3
    - Given Developer has a repo whose `.fitnessrc.json` sets `prose.comments.maxWordsPerToken` or `prose.comments.minBodyComments`
        - When prose-budget checks functions with two long comments
            - Then it uses the configured numbers, and the defaults of 3 and 2 where they are unset

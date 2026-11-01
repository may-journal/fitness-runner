# 0137 Checks Prose in Comments

## Why

My code comments read as easily as my docs, because the prose checks judge them by the same limits. A commit judges only the comments it writes. Each comment says one thing in one statement, so cutting a sentence in two never passes.

## Measurement

Comment blocks within every prose limit
-
Comment blocks judged

## Requirements

- 0137.1
    - Given Developer has a Go, Swift and TypeScript comment with a sentence or a paragraph past its limit
        - When prose-budget checks the repo
            - Then it fails naming each comment's path and line and the limit
- 0137.2
    - Given Developer has a hard-to-read comment in Go, Swift and TypeScript
        - When text-readability checks the repo
            - Then it fails naming each comment's path and line
- 0137.3
    - Given Developer has a comment saying "It's not X. It's Y." in Go, Swift and TypeScript
        - When no-contrastive-reframing checks the repo
            - Then it fails naming each comment's path and line
- 0137.4
    - Given Developer has short why-comments, `//go:build` and `// cspell:ignore` lines, indented code and many short paragraphs in Go, Swift and TypeScript comments
        - When the prose checks run
            - Then they pass, since directives and code are no prose and a comment has no sections
- 0137.5
    - Given Developer stages a change to one comment in a file, named with a space, whose other comment runs on
        - When prose-budget runs on the staged change
            - Then it judges only the changed comment, and a full scan lists both
- 0137.6
    - Given Developer has comments with a "So" sentence, a three-word sentence, or three sentences in a function body
        - When prose-budget checks the repo
            - Then it fails naming each comment's path, line and rule, and says to join, drop or condense it
- 0137.7
    - Given Developer has a 20-word body comment, a three-sentence doc comment, and Markdown with a "So" sentence
        - When prose-budget checks the repo
            - Then it passes, since doc comments keep the paragraph limits and Markdown meets neither rule
- 0137.8
    - Given Developer stages a comment with a "So" sentence in a file whose other comment is a fragment too
        - When prose-budget runs on the staged fragment
            - Then it judges only the changed fragment, and a full scan lists both

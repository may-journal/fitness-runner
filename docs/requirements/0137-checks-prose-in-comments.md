# 0137 Checks Prose in Comments

## Why

My code comments read as easily as my docs, because the prose checks judge them by the same limits. A commit judges only the comments it writes.

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

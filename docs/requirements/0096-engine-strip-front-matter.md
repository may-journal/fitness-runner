# 0096 Strip Front Matter

## Why

No check reads front matter now, so I can drop the old fitness keys from every doc at once. Any other tool's front matter stays where it is.

## Measurement

Markdown files whose fitness-only front matter is gone with the body unchanged
-
Markdown files holding fitness front matter

Source: `go/cmd/fitness/frontmatter.go:93`

## Requirements

- 0096.1
    - Given Developer has docs and entry files whose front matter holds only fitness keys, plain or in an HTML comment
        - When `fitness strip-front-matter` runs
            - Then it removes each block, keeps each body as it was, and lists each file
- 0096.2
    - Given Developer has docs whose front matter also holds another tool's key, like an issue template
        - When `fitness strip-front-matter` runs
            - Then it leaves each file unchanged and names the other keys
- 0096.3
    - Given Developer has docs with no front matter, or front matter without fitness keys
        - When `fitness strip-front-matter` runs
            - Then it leaves them unchanged and unlisted

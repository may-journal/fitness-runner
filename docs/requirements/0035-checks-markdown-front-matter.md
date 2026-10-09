---
relatedConfigurations: ['../../.fitnessrc.json']
---

# 0035 Checks Markdown Front Matter

## Why

Every doc names the checks or configs it explains, so I can jump from a doc to what enforces it.

## Measurement

markdown files whose front matter names real paths or enabled checks
-
markdown files checked

Source: `go/cmd/fitness-check-markdown-front-matter/main.go:83`

## Requirements

- 0035.1
    - Given docs whose first lines carry no front matter with either key
        - When markdown-front-matter checks the repo
            - Then it fails saying each doc's front matter is missing
- 0035.2
    - Given a doc whose front matter key holds an empty array
        - When markdown-front-matter checks the repo
            - Then it fails saying the array must not be empty
- 0035.3
    - Given a doc naming a path that does not exist
        - When markdown-front-matter checks the repo
            - Then it fails naming the missing path
- 0035.4
    - Given a doc in a subfolder naming a path relative to itself
        - When markdown-front-matter checks the repo
            - Then it passes
- 0035.5
    - Given a doc naming a path outside the repo
        - When markdown-front-matter checks the repo
            - Then it fails saying the path escapes the repo
- 0035.6
    - Given a doc naming an enabled check, a web link, an anchor, and a mail link
        - When markdown-front-matter checks the repo
            - Then it passes without resolving them as paths
- 0035.7
    - Given docs inside `node_modules` and `dist` folders
        - When markdown-front-matter checks the repo
            - Then it judges them like any other doc
- 0035.8
    - Given front matter wrapped in an HTML comment on the first line
        - When markdown-front-matter checks the repo
            - Then it reads that front matter and passes

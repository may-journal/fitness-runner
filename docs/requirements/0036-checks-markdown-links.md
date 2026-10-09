---
relatedConfigurations: ['../../.fitnessrc.json']
---

# 0036 Checks Markdown Links

## Why

I can move or delete a doc without leaving broken links behind, because every relative link must reach a real file.

## Measurement

relative links that reach a real file or folder
-
relative links checked

Source: `go/cmd/fitness-check-markdown-links/main.go:102`

## Requirements

- 0036.1
    - Given a doc linking to a missing file
        - When markdown-links checks the repo
            - Then it fails naming the file, line, and link
- 0036.2
    - Given links to an existing file, folder, and heading in another file
        - When markdown-links checks the repo
            - Then it passes
- 0036.3
    - Given a doc in a subfolder linking to a missing sibling
        - When markdown-links checks the repo
            - Then it fails naming the nested file and line
- 0036.4
    - Given an image or reference definition pointing at a missing file
        - When markdown-links checks the repo
            - Then it fails naming each one
- 0036.5
    - Given web, mail, protocol-relative, and same-page heading links
        - When markdown-links checks the repo
            - Then it passes, since it never goes online
- 0036.6
    - Given a broken link inside a code block or inline code
        - When markdown-links checks the repo
            - Then it passes, since code is not a link
- 0036.7
    - Given a percent-encoded link to a file with a space in its name
        - When markdown-links checks the repo
            - Then it passes

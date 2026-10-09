---
relatedConfigurations: ['../../.fitnessrc.json']
---

# 0037 Checks Markdown No Bold Italic

## Why

I keep docs plain, so emphasis never stands in for clear words. Any bold or italic text gets flagged before it lands.

## Measurement

docs free of bold and italic emphasis
-
docs checked

Source: `go/cmd/fitness-check-markdown-no-bold-italic/main.go:163`

## Requirements

- 0037.1
    - Given a doc with asterisk bold and italic
        - When markdown-no-bold-italic checks it
            - Then it fails quoting each emphasized snippet
- 0037.2
    - Given a doc with underscore bold and italic
        - When markdown-no-bold-italic checks it
            - Then it fails quoting each emphasized snippet
- 0037.3
    - Given emphasis only inside inline or fenced code
        - When markdown-no-bold-italic checks the doc
            - Then it passes, since code is not prose
- 0037.4
    - Given underscores inside a markdown link
        - When markdown-no-bold-italic checks the doc
            - Then it passes
- 0037.5
    - Given a list with `*` item markers
        - When markdown-no-bold-italic checks the doc
            - Then it passes
- 0037.6
    - Given a CHANGELOG with bold text
        - When markdown-no-bold-italic runs
            - Then it fails naming `CHANGELOG.md`
- 0037.7
    - Given an issue or PR body with italic text
        - When markdown-no-bold-italic checks it in body mode
            - Then it fails naming the description

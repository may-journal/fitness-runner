---
relatedConfigurations: ['../../.fitnessrc.json']
---

# 0037 Checks Markdown No Bold Italic

## Why

I keep docs plain, so emphasis never stands in for clear words. Any bold or italic text gets flagged before it lands. Release Please notes pass, since only their scope is bold.

## Measurement

docs free of bold and italic emphasis
-
docs checked

Source: `go/cmd/fitness-check-markdown-no-bold-italic/main.go:163`

## Requirements

- 0037.1
    - Given Developer has a doc with asterisk bold and italic
        - When markdown-no-bold-italic checks it
            - Then it fails quoting each emphasized snippet
- 0037.2
    - Given Developer has a doc with underscore bold and italic
        - When markdown-no-bold-italic checks it
            - Then it fails quoting each emphasized snippet
- 0037.3
    - Given Developer has emphasis only inside inline or fenced code
        - When markdown-no-bold-italic checks the doc
            - Then it passes, since code is not prose
- 0037.4
    - Given Developer has underscores inside a markdown link
        - When markdown-no-bold-italic checks the doc
            - Then it passes
- 0037.5
    - Given Developer has a list with `*` item markers
        - When markdown-no-bold-italic checks the doc
            - Then it passes
- 0037.7
    - Given Developer has an issue or PR body with italic text
        - When markdown-no-bold-italic checks it in body mode
            - Then it fails naming the description
- 0037.9
    - Given Developer has a CHANGELOG with bold text beside Release Please bullets' bold scopes
        - When markdown-no-bold-italic runs
            - Then only that bold fails, naming `CHANGELOG.md`

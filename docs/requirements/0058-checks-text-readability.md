---
relatedConfigurations: ['../../.fitnessrc.json']
---

# 0058 Checks Text Readability

## Why

I catch run-on docs and notation posing as prose, without being punished for ordinary dense writing.

## Measurement

docs with 2 of 3 scores in alarm reported red
-
docs with 2 of 3 scores in alarm

Source: `go/cmd/fitness-check-text-readability/main.go:143`

## Requirements

- 0058.1
    - Given a doc of long words in one endless sentence
        - When text-readability checks the repo
            - Then it fails naming the doc, its scores, and how to fix it
- 0058.2
    - Given a doc of plain short sentences
        - When text-readability checks the repo
            - Then it passes
- 0058.3
    - Given a dense doc under 100 prose words
        - When text-readability checks the repo
            - Then it passes, since short docs are not judged
- 0058.4
    - Given a doc where only one score reaches its alarm
        - When text-readability checks the repo
            - Then it passes, since 2 of 3 must agree
- 0058.5
    - Given dense notation inside a fenced code block
        - When text-readability checks the repo
            - Then it passes, since code is not scored
- 0058.6
    - Given a dense issue or PR body
        - When text-readability checks it in body mode
            - Then it fails naming the description
- 0058.7
    - Given a repo that lowers its own `textReadability` bands
        - When text-readability checks plain prose
            - Then it fails against those bands
- 0058.8
    - Given the `--report` flag
        - When text-readability checks the repo
            - Then it prints each doc's word count without changing the verdict

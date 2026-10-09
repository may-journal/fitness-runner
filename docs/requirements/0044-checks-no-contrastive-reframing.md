---
relatedConfigurations: ['../../.fitnessrc.json']
---

# 0044 Checks No Contrastive Reframing

## Why

I want docs that state the point directly, without first rejecting a claim nobody made.

## Measurement

markdown files with no reframed sentence
-
markdown files checked

Source: `go/cmd/fitness-check-no-contrastive-reframing/main.go:59`

## Requirements

- 0044.1
    - Given a doc with a negated sentence followed by a restating sentence
        - When no-contrastive-reframing checks the doc
            - Then it fails naming the file and quoting both sentences
- 0044.2
    - Given a doc with one sentence that negates, then pivots with a comma
        - When no-contrastive-reframing checks the doc
            - Then it fails quoting that sentence
- 0044.3
    - Given a doc with ordinary negations that restate nothing
        - When no-contrastive-reframing checks the doc
            - Then it passes
- 0044.4
    - Given the pattern in front matter, fenced code, a heading, or inline code
        - When no-contrastive-reframing checks the doc
            - Then it passes, since that text is not prose
- 0044.5
    - Given the pattern inside straight or curly quotes
        - When no-contrastive-reframing checks the doc
            - Then it passes, so a doc can cite the pattern
- 0044.6
    - Given a reframed sentence longer than 80 characters
        - When no-contrastive-reframing checks the doc
            - Then it quotes the first 80 characters and an ellipsis

---
relatedConfigurations: ['../../.fitnessrc.json']
---

# 0020 Checks Commit Attribution

## Why

I can see which AI tools and models helped write each commit. Repos that never adopted the convention are left alone.

## Measurement

commit messages carrying both AI trailers
-
commit messages checked in repos that adopted the trailers

Source: `go/cmd/fitness-check-commit-attribution/main.go:123`

## Requirements

- 0020.1
    - Given a folder with no git history, or no recent commit with an AI trailer
        - When commit-attribution runs
            - Then it passes having checked 0 files
- 0020.2
    - Given a last commit with `AI-Tools:` and `AI-Models:` trailers
        - When commit-attribution runs
            - Then it passes
- 0020.3
    - Given an adopting repo whose last commit has no AI trailers
        - When commit-attribution runs
            - Then it fails naming each missing trailer
- 0020.4
    - Given a proposed message without trailers and a last commit with both
        - When commit-attribution runs with `--message`
            - Then it fails, judging the proposed message
- 0020.5
    - Given an adopting repo and an empty proposed message
        - When commit-attribution runs with `--message`
            - Then it fails saying there is no message to validate
- 0020.6
    - Given an adopting repo whose last commit is a merge or revert
        - When commit-attribution runs
            - Then it passes, since those commits are exempt
- 0020.7
    - Given an `AI-Models:` trailer with an empty value
        - When commit-attribution runs
            - Then it fails naming that trailer as missing
- 0020.8
    - Given `AI-Models:` mentioned in the middle of a sentence
        - When commit-attribution runs
            - Then it fails naming that trailer as missing

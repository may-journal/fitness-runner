---
relatedConfigurations: ['../../.fitnessrc.json']
---

# 0051 Checks PR Structure

## Why

I want every PR description to give a summary, the reason, and changelog entries. The diff already shows what changed.

## Measurement

PR bodies off the template reported red
-
PR bodies off the template

Source: `go/cmd/fitness-check-pr-structure/main.go:29`

## Requirements

- 0051.1
    - Given Developer has a PR body with a summary, a reason, and a changelog bullet
        - When fitness runs pr-structure on it
            - Then it reports green
- 0051.2
    - Given Developer has a PR body with no blockquote summary
        - When fitness runs pr-structure on it
            - Then it reports red asking for the summary
- 0051.3
    - Given Developer has a PR body whose summary is still the template placeholder
        - When fitness runs pr-structure on it
            - Then it reports red asking for the real summary
- 0051.4
    - Given Developer has a PR body without a `## Background` section
        - When fitness runs pr-structure on it
            - Then it reports red naming the missing section
- 0051.5
    - Given Developer has a PR body whose `## Changelog` has no bullet
        - When fitness runs pr-structure on it
            - Then it reports red asking for a list item
- 0051.6
    - Given Developer has a PR body with an extra `## Testing` section
        - When fitness runs pr-structure on it
            - Then it reports red naming the section and the allowed ones
- 0051.7
    - Given Developer has a PR body whose extra heading sits inside fenced code
        - When fitness runs pr-structure on it
            - Then it reports green
- 0051.8
    - Given Developer has no PR body at all
        - When fitness runs pr-structure
            - Then it passes with zero files checked

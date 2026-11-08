# 0147 Checks ADR As Issue

## Why

I keep architecture decisions in `ADR` issues people can discuss. A decision never hides in a file that drifts from its talk.

## Measurement

ADR files reported red
-
ADR files in the repo

## Requirements

- 0147.1
    - Given Developer keeps ADR files in an `adr` folder beside its README
        - When fitness runs adr-as-issue
            - Then it fails naming each ADR file and telling me to open an `ADR` issue, and passes the README
- 0147.2
    - Given Developer has a Markdown file titled `ADR 12`
        - When fitness runs adr-as-issue
            - Then it fails naming that file
- 0147.3
    - Given Developer has a check README that still holds its Context, Decision and Consequences
        - When fitness runs adr-as-issue
            - Then it fails naming the README, while the same sections in a code block pass
- 0147.4
    - Given Developer keeps every ADR as an issue, linked from its old file and its check README
        - When fitness runs adr-as-issue
            - Then it passes
- 0147.5
    - Given Developer allows the files still waiting to move, each with a reason
        - When fitness runs adr-as-issue
            - Then it passes the allowed files and fails the rest
- 0147.6
    - Given Developer has an allow entry with no reason, or one that matches no ADR file
        - When fitness runs adr-as-issue
            - Then it fails naming the entry and what to fix

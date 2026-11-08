# 0148 ADR Issues

## Why

Every `ADR` issue states its decision the same way, so I read any of them at a glance.

## Measurement

ADR issue edits whose verdict matches the ADR rules
-
ADR issue edits

## Requirements

- 0148.1
    - Given Developer has an `ADR` issue with a `[ADR]` title, a one-word Status, Context, Decision and Consequences
        - When issue-check judges it
            - Then it comments that the ADR looks good
- 0148.2
    - Given Developer has an `ADR` issue with a plain title, a long Status, or a missing Consequences section
        - When issue-check judges it
            - Then it fails naming each problem

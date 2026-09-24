---
relatedConfigurations: ['../../../.fitnessrc.json']
---

# plan-trailer

Validates an optional `Plan #NN` commit trailer. A commit may name the Plan Issue it implements with a line reading exactly `Plan #<number>`. The trailer is optional, so a message without one passes. A plan reference that is present must be well formed.

Like `semantic-commit`, it reads the proposed message from the context-inline `--message` value, falling back to the HEAD commit message. The `commit-msg` hook runs it beside `semantic-commit`.

## Behavior

- Pass: the message has no plan-reference line, or its reference reads exactly `Plan #<number>`.
- Fail: a plan-reference line in another spelling, one error each. `Plan 63`, `plan #63`, and `Plan: 63` all fail.
- Body prose is never a trailer. A line like "Plan the rollout in 3 steps" is ignored.
- `filesChecked` is always 1 (the one commit message).

## Contributing

This README is the canonical description for this check, a self-contained `fitness-check-plan-trailer` binary. To change the accepted form, extend the check and its tests here and keep this README in sync.

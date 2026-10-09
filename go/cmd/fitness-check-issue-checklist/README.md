---
relatedConfigurations: ['../../../.fitnessrc.json']
---

# issue-checklist

Fails an issue body whose task list still has unchecked items. Closing such an issue would close unfinished work.

It reads the body from a `--body-file` path, `-` meaning stdin, or from the runner's context-inline `--body` value. With no input it passes with zero files checked. `fitness close-check` runs it on a closed issue, and `fitness pr-check` on each issue a pull request closes.

## Behavior

- Pass: every `- [ ]` item is ticked, or the body has no task list.
- Fail: one error per unchecked item, quoting its text.

Items inside fenced code are examples, so they never count.

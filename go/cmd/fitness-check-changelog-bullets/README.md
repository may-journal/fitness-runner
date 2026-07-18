---
fitnessFunctions: ['main.go']
relatedConfigurations: ['../../../.fitnessrc.json']
---

# changelog-bullets

Keeps changelog entries tight: the first (newest) section of `CHANGELOG.md` must be 3 to 5 bullets, each under 365 characters, each starting with a capitalized semantic type prefix.

## Behavior

- Only the first `###` section is judged — older sections are history.
- Bullet count must be 3 to 5; continuation lines join their bullet for the length count.
- Every bullet must start with one of: `Feat:`, `Fix:`, `Docs:`, `Style:`, `Refactor:`, `Perf:`, `Test:`, `Build:`, `Ci:`, `Chore:`, `Revert:`.
- A repo without `CHANGELOG.md` passes with zero files checked.

## Errors

```text
CHANGELOG.md first section has 1 bullets; keep entries to 3-5 bullets
CHANGELOG.md:12: bullet is 1230 characters; keep each under 365
CHANGELOG.md:14: bullet must start with a semantic type (Feat:, Fix:, ...)
```

## Enable

Opt-in — add the name to your `checks` list in `.fitnessrc.json`:

```json
{
  "checks": ["changelog-bullets"]
}
```

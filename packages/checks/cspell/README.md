---
fitnessFunctions: ['cspell']
relatedConfigurations: ['../../shared/config/cspell.json']
---

# cspell

Runs [cspell](https://cspell.org) for spell-checking. Uses a local `cspell.json` when present; otherwise uses `@mayjournal/fitness-shared/cspell`. Cspell is bundled as a dependency—consumers do not install it separately.

## Behavior

- Pass: No unknown words in checked files.
- When context has staged files: Only those paths are checked.
- Otherwise: Runs cspell on `**/*.md`.
- Config: Local `cspell.json` if present; otherwise `@mayjournal/fitness-shared/cspell` (no copy required in the consumer repo).

Errors are reported as one line per issue: `path:line:col - Unknown word (word)`.

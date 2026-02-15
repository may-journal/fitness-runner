---
fitnessFunctions: ["cspell"]
relatedConfigurations: ["../../../cspell.json"]
---

# cspell

Runs [cspell](https://cspell.org) for spell-checking. The runner lists cspell as a dependency so it’s available when you use this check. Optional: the check only runs when you have a `cspell.json` at repo root; if missing, the check passes without running.

## Behavior

- Pass: No unknown words in checked files.
- When context has staged files: Only those paths are checked.
- Otherwise: Runs cspell on `**/*.md`.
- Skip: No `cspell.json` at repo root → pass with 0 files checked (check is effectively off).

Errors are reported as one line per issue: `path:line:col - Unknown word (word)`.

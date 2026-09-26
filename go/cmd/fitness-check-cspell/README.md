---
fitnessFunctions: ['cspell']
relatedConfigurations: ['../../internal/sharedconf/config/cspell.json']
---

# cspell

Spell-checks with a native Go engine and dictionaries embedded in the check binary — no external cspell tool, no npm install. The `cspell.json` words and ignore settings still apply.

## Behavior

- Pass: No unknown words in checked files.
- When context has staged files: Only those paths are checked.
- Otherwise: Runs the spell-check on `**/*.md`.
- Config: A repo-local `cspell.json` wins; otherwise an installed `@mayjournal/fitness-shared` package under `node_modules`; otherwise the copy embedded in the binary, materialized on demand.
- No copy is required in the consumer repo.

Errors are reported as one line per issue: `path:line:col - Unknown word (word)`.

---
fitnessFunctions: ['cspell']
relatedConfigurations: ['../../internal/sharedconf/config/cspell.json']
---

# cspell

Spell-checks with a native Go engine and dictionaries embedded in the check binary — no external cspell tool, no npm install. The `cspell.json` words still apply; it ignores no path.

## Behavior

- Pass: No unknown words in checked files.
- In a scoped run: Only changed paths are checked.
- Otherwise: Spell-checks every tracked file; binary files count but are never spelled.
- Generated files: a file `.gitattributes` marks `linguist-generated`, such as `package-lock.json`, is left out; no other skip exists.
- Fail: A resolved `cspell.json` that sets `ignorePaths` fails before any scan; remove it and fix the words instead.
- Config: A repo-local `cspell.json` wins; otherwise an installed `@mayjournal/fitness-shared` package under `node_modules`; otherwise the copy embedded in the binary, materialized on demand.
- No copy is required in the consumer repo.

Errors are reported as one line per issue: `path:line:col - Unknown word (word)`.

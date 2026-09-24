---
fitnessFunctions: ['read-repo-first']
---

# read-repo-first

Displays feedback to the CLI reminding agents/users to familiarize themselves with decisions logged in the repo and all fitness checks enabled for it.

## Behavior

- Always passes; no TTY or interactive prompt.
- Outputs a boxed message to stderr; stdout carries the JSON result.
- The box shows the question, a table of the enabled check names when the runner provides them, and a note about not using `--no-verify`.

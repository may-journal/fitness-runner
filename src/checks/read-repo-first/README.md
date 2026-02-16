---
fitnessFunctions: ["read-repo-first"]
---

# read-repo-first

Displays feedback to the CLI reminding agents/users to familiarize themselves with decisions logged in the repo and all Fitness Checks enabled via fitness-runner.

## Behavior

- Always passes; no TTY or interactive prompt.
- Outputs a boxed message to stdout with the question, enabled check names, and a note about not using `--no-verify`.

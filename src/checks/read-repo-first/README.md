---
fitnessFunctions: ["read-repo-first"]
---

# read-repo-first

Prompts the user to confirm they familiarized themselves with decisions logged in the repo and all Fitness Checks enabled via fitness-runner.

## Behavior

- Interactive (TTY): Asks if you familiarized yourself with decisions and enabled checks, lists check names, Y or N.
- Pass: User answers Y or y.
- Fail: User answers N, n, or anything else.
- Non-interactive (no TTY): Fail with "Run fitness interactively to complete this check (requires TTY)."
- Bypass: Set `FITNESS_READ_REPO_CONFIRMED=1` to pass without prompting (e.g. in CI after prior review).

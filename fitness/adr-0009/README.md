# 0009 — Repos add local checks; only external policy replaces or disables a bundled one

A repo adds checks, and the org bar stays whole.

https://github.com/may-journal/fitness-runner/issues/861

## Check

This check keeps the decision in fitness itself.

- Fails `.fitnessrc.json` when it sets `disabledChecks` or external policy.
- Fails a check in `fitness/` named like a bundled check, which would replace it.
- Fix: fix the findings, or give the new check a name of its own.

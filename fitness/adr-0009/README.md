# 0009 — Repos add local checks; only external policy replaces or disables a bundled one

A repo adds checks, and the org bar stays whole.

https://github.com/may-journal/fitness-runner/issues/861

## Check

This check keeps the decision in fitness itself.

- Fails `.fitnessrc.json` when it sets `disabledChecks` or external policy.
- Fix: keep org policy and fix the findings instead.

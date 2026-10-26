# ADR 0009 check

Keeps [ADR 0009](../../docs/architecture/adr/0009-repos-add-checks-org-keeps-the-bar.md) in fitness itself: a repo adds checks, and the org bar stays whole.

- Fails `.fitnessrc.json` when it sets `disabledChecks` or external policy.
- Fails a check in `fitness/` named like a bundled check, which would replace it.
- Fix: fix the findings, or give the new check a name of its own.

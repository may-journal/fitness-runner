---
relatedConfigurations: ['../../../.fitnessrc.json']
---

# no-plans-dir

Guards the plans-to-Issues migration. Plans now live as `Plan`-labeled GitHub Issues, not checked-in files, so this check fails when any file exists under `docs/plans/`. It points the author to open a Plan Issue instead.

Wired into the file suite, so the pre-commit hook catches a `docs/plans/` file before it lands.

## Behavior

- Pass: no file exists under `docs/plans/`.
- Fail: one error per file found there, naming the file and the fix.
- `filesChecked` is the number of offending files (zero when clean).

## Contributing

This README is the canonical description for this check, a self-contained `fitness-check-no-plans-dir` binary. To change the guarded path, extend the check and its test here and keep this README in sync.

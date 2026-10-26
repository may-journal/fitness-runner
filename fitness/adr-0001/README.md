# ADR 0001 check

Keeps [ADR 0001](../../docs/architecture/adr/0001-fix-the-work-not-the-limit.md): a check earns its budget, and a slow check gets its work cut.

- Fails a built-in check whose declared `TimeoutMs` has no entry in [config.json](config.json) naming the external cost behind it.
- Fails an entry in `config.json` that no longer matches its check.
- Fails a `timeoutMs` key in `.fitnessrc.json`, which would raise every check's budget.
- Fix: cut the work, or add the check's entry with the network round-trip or tool run it waits on.

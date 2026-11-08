### 2026.11.08.1800

- Perf: `workflow-ref-exists` verifies up to eight `uses:` calls at once, sharing each GitHub read, instead of one at a time.
- Fix: after the first failure to reach GitHub, `workflow-ref-exists` starts no new call, as the serial check did.
- Docs: the `workflow-ref-exists` page says it verifies calls side by side and reads each GitHub path once.
- Test: `Test0086_8` sees eight calls, each read once after one second, pass in about one second instead of eight.

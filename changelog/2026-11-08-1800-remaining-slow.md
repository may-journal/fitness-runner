### 2026.11.08.1800

- Perf: `workflow-ref-exists` verifies up to eight `uses:` calls at once, sharing each GitHub read, instead of one at a time.
- Docs: the `workflow-ref-exists` page says it verifies calls side by side and reads each GitHub path once.
- Test: `Test0086_8` sees eight calls answered after one second each pass in about one second instead of eight.

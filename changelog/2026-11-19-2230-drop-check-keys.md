### 2026.11.19.2230

> The engine stops declaring the keys checks read, so each check's own `--describe` is the one source of its `.fitnessrc.json` keys.

- Refactor: `conf.Config` keeps only the engine's keys, and `conf.MaxListItems` goes, since nothing calls it.
- Refactor: `pkg/adr` and `pkg/lang/golang` drop their config sections, and `lang.Module.Scans` and `lang.ScanExts` go.
- Fix: `fitness config schema` also lists the keys of checks that run only when a repo names them, such as `goTestCoverage`.

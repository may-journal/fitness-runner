# 0008 — Languages plug in as modules through one registry

Each language plugs in as one module.

https://github.com/may-journal/fitness-runner/issues/860

## Check

- Reads each file in `go/internal/lang/langs` beside the registry in `langs.go`.
- Fails a file that declares no `lang.Module` value, or more than one.
- Fix: give each language its own file with its one module.

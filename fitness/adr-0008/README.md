# ADR 0008 check

Keeps [ADR 0008](../../docs/architecture/adr/0008-languages-plug-in-as-modules.md): each language plugs in as one module.

- Reads each file in `go/internal/lang/langs` beside the registry in `langs.go`.
- Fails a file that declares no `lang.Module` value, or more than one.
- Fix: give each language its own file with its one module.

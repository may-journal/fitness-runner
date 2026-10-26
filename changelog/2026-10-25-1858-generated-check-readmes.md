### 2026.10.25.1858

- Docs: each check's rules move into its package doc comment, and `go generate` writes its README from it, the same text `fitness explain` prints.
- Docs: `go generate` writes the check index in docs/checks.md and the README's command list, which `fitness --help` prints from the same constant.
- fix(requirements): a requirement doc whose Source cites another line of its file, as when the code moved, needs no test change.
- Test: requirement 0114 compares each README, the index and the command list with what the binary prints.
- Fix: a scoped run of the generated check still judges a change to a README or index it writes.

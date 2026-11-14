### 2026.11.14.1730

- Refactor: 34 shared package folders move from `go/internal` to `go/pkg`, so another Go module can import them; no logic changes.
- Refactor: the eight check-only `lang` leaves, `spell`, `sharedconf`, `architecture`, `clonedetect`, `mdfilename` and `requirement` stay internal.
- Docs: each `fitness docs` topic prints the doc comments of its package under `go/pkg` or `go/internal`, as requirement 0112 says.
- Fix: a build holding `local-branch` without the engine fails with a message naming fitness, instead of panicking.
- Test: requirement 0160 proves an outside module builds against every package under `go/pkg`.

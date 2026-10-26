### 2026.10.26.2125

- Feat: each `fitness/<name>/` folder is a repo's own check: a README and Go source that `fitness check <name>` builds once and runs.
- Feat: the `go/check` SDK gives a Go check typed inputs, config and findings, and `check.Repo` and `check.Judge` unit-test it.
- Feat: the new `adr-checks` pairs each ADR with its `fitness/adr-NNNN/` check, and `requirements` accepts a `fitness/req-NNNN/` check as proof.
- Feat: an ADR changed without its check fails, and a paired check that judges no files fails as a broken scan.
- Feat: fitness keeps its own Go checks, one per ADR, in `fitness/`, as ADR 0012 records; any executable stays the fallback.

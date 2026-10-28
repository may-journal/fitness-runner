### 2026.10.26.2125

- Feat: each `fitness/<name>/` folder is a repo's own check: a README and Go source that `fitness check <name>` builds once and runs.
- Feat: the `go/check` SDK gives a Go check typed inputs, config and findings, and `check.Repo` and `check.Judge` unit-test it.
- Feat: each ADR lives in its `fitness/adr-NNNN/` check's README, where docs link it, and `adr-checks` fails one kept elsewhere or lacking its decision.
- Feat: changing an ADR's decision without its check's code fails, as does a paired check that judges no files.
- Feat: `requirements` accepts a `fitness/req-NNNN/` check as proof, and fitness keeps its own checks, one per ADR, in `fitness/`.

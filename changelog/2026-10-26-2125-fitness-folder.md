---
relatedConfigurations: ['../.fitnessrc.json', '../docs/requirements/0131-engine-fitness-folder.md', '../docs/requirements/0132-checks-paired-checks.md']
---

### 2026.10.26.2125

- Feat: each `fitness/<name>/` folder is a repo's own check: a README, and an executable `check` or Go source fitness builds.
- Feat: the public `go/check` SDK writes a Go check; `fitness check <name>` runs one alone, and `fitness list` and `explain` read its README.
- Feat: the new `adr-checks` pairs each ADR with its `fitness/adr-NNNN/` check, and `requirements` accepts a `fitness/req-NNNN/` check as proof.
- Feat: a doc or its check changed alone fails, and a paired check that judges no files fails as a broken scan.
- Feat: fitness keeps its own checks, one per ADR, in its `fitness/` folder, as ADR 0012 records.

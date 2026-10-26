---
relatedConfigurations: ['../.fitnessrc.json', '../docs/requirements/0131-engine-fitness-folder.md']
---

### 2026.10.26.2125

- Feat: each `fitness/<name>/` folder is a repo's own check, with a README and an executable `check` or Go source, and needs no config.
- Feat: fitness builds a Go check with `fitness/go.mod` into its cache once, and names where to get Go when it is missing.
- Feat: the public `go/check` SDK writes a check in Go, and `check.Test` runs it as a Go test.
- Feat: `fitness list` shows a local check with scope local and its README's summary, `fitness explain` prints the README, and `minFitness` names the upgrade.
- Test: requirement 0131 runs shell and Go checks, cold and warm, through the installed command.

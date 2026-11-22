### 2026.11.15.0710

> Every workflow and action step runs a typed Go subcommand, and no step runs a shell script.

- Refactor: `go/cmd/fitness-ci` runs each step as one subcommand, sharing `go/pkg/actions` for outputs, gh and git.
- Feat: the release bundle ships fitness-ci beside fitness, so repos that use the actions need no Go.
- Feat: the Fitness action downloads its installer with `gh release download`, which refuses a release outside a major ref.
- Feat: `make fitness-ci` builds it into `bin/` for this repo's jobs, and `fitness-release verify-tag` writes the version output.
- Test: requirements 0008.7 and 0008.8 run each subcommand as its workflow does and check the outputs it declares.

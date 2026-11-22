### 2026.11.15.0710

> Every workflow and action step that did real work in shell now calls a typed Go subcommand.

- Refactor: `go/cmd/fitness-ci` runs each multi-line step as one subcommand, sharing `go/pkg/actions` for outputs, gh and git.
- Feat: `.github/actions/fitness-ci` builds the program once per job and runs a subcommand for reusable workflows and shared actions.
- Feat: `fitness-release verify-tag` writes the release version as a step output.
- Chore: CI runs shellcheck over the scripts that stay in shell.
- Test: requirements 0008.7 and 0008.8 run each subcommand as its workflow does and check the outputs it declares.

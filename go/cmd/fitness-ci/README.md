# fitness-ci

The steps of this repo's workflows and shared actions, as one typed Go program. Each step that does real work calls one subcommand on one line. The subcommands share the helpers in [go/pkg/actions](../../pkg/actions), and requirement 0008.7 runs each one the way its workflow does.

## Build once per job

A job in this repo builds it into `bin/`, which git ignores. Each later step calls `bin/fitness-ci <subcommand>`:

```bash
go -C go build -o ../bin/fitness-ci ./cmd/fitness-ci
bin/fitness-ci find-go-module
```

Reusable workflows and shared actions run in other repos, whose checkouts do not hold this program. They call the [fitness-ci action](../../../.github/actions/fitness-ci/action.yml) with the subcommand as its `command` input. Its first use in a job sets up Go and builds the program into the action checkout's `bin/` folder. It then sets `FITNESS_CI`, so later uses skip the build.

The action declares every step output a subcommand writes.

## Subcommands

`fitness-ci help` lists every subcommand, what it does and the step outputs it writes. A subcommand reads its inputs from the step's environment, as the shell steps it replaces did. It writes outputs, environment and the job summary through the files Actions names. Release logic stays in [fitness-release](../fitness-release/README.md), which the release steps here run.

## What stays in shell

The root [action.yml](../../../action.yml) runs a verified bundle without compiling, even when a caller keeps Go off `PATH`. So its download and report steps stay in `scripts/`. The `fitness-install` and `save-checks` steps on that path, which predate this program, stay in shell too. CI lints them with shellcheck.

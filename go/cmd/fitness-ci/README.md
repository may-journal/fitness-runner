# fitness-ci

The steps of this repo's workflows and actions, as one typed Go program. Each step that does real work calls one subcommand on one line, and no step runs a shell script. The subcommands share the helpers in [go/pkg/actions](../../pkg/actions), and requirement 0008.7 runs each one the way its workflow does.

## This repo's jobs

A job here builds it from the checkout into `bin/`, which git ignores, and calls `bin/fitness-ci <subcommand>`:

```bash
make fitness-ci
bin/fitness-ci find-go-module
```

The [setup-go-cache action](../../../.github/actions/setup-go-cache/action.yml) runs `make fitness-ci` and puts `bin/` on `PATH`, so the shared actions a job uses find it too.

## Other repos

Repos that use the actions need no Go. fitness-ci ships in the release bundle beside `fitness`, so a job that installed fitness already has it on `PATH`. The [fitness-ci action](../../../.github/actions/fitness-ci/action.yml) runs a subcommand, installing the bundle first in a job that has none. Until a release holds fitness-ci, it builds the program from its own checkout instead.

The root [action.yml](../../../action.yml) downloads the installer with `gh release download`, before any fitness is installed. The installer checks the bundle against the hashes it holds, and a major action ref against its own version.

## Subcommands

`fitness-ci help` lists every subcommand, what it does and the step outputs it writes. A subcommand reads its inputs from the step's environment and writes outputs, environment and the job summary through the files Actions names. Release logic stays in [fitness-release](../fitness-release/README.md), which the release steps here run.

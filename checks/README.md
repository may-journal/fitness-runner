# Local checks

This repo's own checks, which `.fitnessrc.json` names with `"localChecks": "checks"`. Each runs in the fitness suite, the git hooks and CI, as the [check protocol](../docs/check-protocol.md#local-checks) describes.

## generated

`fitness-check-generated` fails when a file `go generate` writes is stale or edited by hand. It also fails when the generator cannot read the code, such as a check package with no doc comment.

- Usage: `fitness generated`, or as part of `fitness`.
- Inputs: the Go packages under `go/cmd/fitness-check-*`, and their doc comments.
- Outputs: the check's JSON result, with one error per stale file.
- Fix: run `go -C go generate ./cmd/fitness` and commit what it writes.

The program answers `--describe` itself, since compiling `fitness-release` can outlast the handshake's budget. The check fails when that line drifts from the Go copy. Every other run goes to `fitness-release generated`, which compares the files on disk with what the generator writes now. A scoped run that changes no Go file skips.

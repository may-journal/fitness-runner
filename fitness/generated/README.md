# generated

Fails when a file `go generate` writes is stale or edited by hand. It also fails when the generator cannot read its sources, such as a topic without its title and summary sentence.

- Usage: `fitness generated`, or as part of `fitness`.
- Inputs: the Markdown in `go/docs`, the usage comments of `go/cmd/fitness`, and the JSON tags of `Config` in `go/pkg/conf`.
- Outputs: the check's JSON result, with one error per stale file.
- Fix: run `go -C go generate ./cmd/fitness` and commit what it writes.

The program answers `--describe` itself, since compiling `fitness-release` can outlast the handshake's budget. The check fails when that line drifts from the Go copy. Every other run goes to `fitness-release generated`, which compares the files on disk with what the generator writes now. A scoped run that changes no Go file, nothing in `go/docs` and no file it writes skips.

It is one of this repo's [local checks](../../docs/local-checks.md), as [ADR 0012](https://github.com/may-journal/fitness-runner/issues/864) decides.

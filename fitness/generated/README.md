# generated

Fails when a file `go generate` writes is stale or edited by hand. It also fails when the generator cannot read the code, such as a check package with no doc comment. It fails a Go package that names no container or component in its doc comment, or names one the views do not draw.

- Usage: `fitness generated`, or as part of `fitness`.
- Inputs: the Go packages under `go/cmd/fitness-check-*` and their doc comments, and the `usage` constant of `go/cmd/fitness`.
- Views: the `//fitness:container` and `//fitness:component` lines of every Go package, and the elements the views draw.
- Outputs: the check's JSON result, with one error per stale file.
- Fix: run `go -C go generate ./cmd/fitness` and commit what it writes.

The program answers `--describe` itself, since compiling `fitness-release` can outlast the handshake's budget. The check fails when that line drifts from the Go copy. Every other run goes to `fitness-release generated`, which compares the files on disk with what the generator writes now. A scoped run that changes no Go file and no file it writes skips.

It is one of this repo's [local checks](../../docs/local-checks.md), as [ADR 0012](../adr-0012/README.md) decides.

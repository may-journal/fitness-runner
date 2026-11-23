# local-checks

go/pkg/repocheck finds a repo's own checks and builds the Go ones, as ADR 0009, ADR 0012 and docs/local-checks.md describe.

## ADRPrefix

ADRPrefix starts the name of a check folder whose README.md is an architecture decision record, beside the check that keeps it.

## Folder

Folder is the folder at a repo's root that holds its own checks, read with no config.

## Program

Program is the file name of a folder check's executable entry point.

## Readme

Readme is the file every folder check documents its rule in.

## Check

Check is one of a repo's own checks.

Fields:

- `Name` is the check name, from the folder or program name.
- `Path` is the program to run. It is empty for a Go check until Build writes one.
- `Shown` is where the check lives, relative to the repo root, as a person reads it in a message.
- `Folder` is a folder check's folder relative to the repo root, or empty for a flat program.
- `Readme` is the check's README.md, or empty for a flat program.
- `Source` is the folder of Go source to build, or empty when Path is the program.
- `Module` is the folder holding the go.mod Source builds with.

## Build

Build returns the cached program for a Go check or builds one, keyed by its sources, Go version, platform and fitness version.

## Doc

Doc returns a folder check's README text after its title line, as `fitness explain` prints it, or "" when unreadable.

## Find

Find maps by name each check in root's fitness/ and configured localChecks folders, failing on an empty folder or duplicate name.

## NeedsNewer

NeedsNewer says why the running fitness is too old for a check's minFitness, or "" when new enough or unversioned.

## Paired

Paired reports whether a check of this name keeps an ADR or proves a requirement, where judging no files means a broken scan.

## Retired

Retired reports whether an ADR's text says it is superseded, so it needs no check of its own.

## Summary

Summary is the first sentence of the README's first paragraph, which `fitness list` shows for a folder check whose --describe gives none.

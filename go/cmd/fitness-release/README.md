---
relatedConfigurations: ['../../../.github/workflows/release.yml', '../../../.goreleaser-bundles.yml', '../../../.goreleaser-tools.yml']
---
<!-- cspell:ignore goreleaser -->


# fitness-release

Small Go helpers for bundle hashes, native checks, release promotion, and pin edits. Release Please owns versions, tags, and release PRs. GoReleaser owns builds, archives, checksums, and publication.

## Build a local snapshot

From the repository root, with GoReleaser OSS 2.18.2 installed:

```bash
export FITNESS_VERSION="$(go -C go run ./cmd/fitness-release version --root ..)"
goreleaser release --clean --snapshot --skip=publish --config .goreleaser-bundles.yml
export FITNESS_BUNDLE_HASHES="$(go -C go run ./cmd/fitness-release bundle-hashes --root ..)"
goreleaser release --clean --snapshot --skip=publish --config .goreleaser-tools.yml
go -C go run ./cmd/fitness-release assemble --root ..
```

The bundle pass creates four archives. The tools pass embeds their hashes in four installers and builds native verifier tools. `assemble` copies the outputs into `out` and writes metadata for the checks. Verifiers remain CI artifacts; consumers only need the installer.

## Commands

| Command | Purpose |
| --- | --- |
| `version` | Print the version from `version.txt`. |
| `tag` | Print its root `v` tag. |
| `checks` | Write `go/cmd/fitness/checks.go`, the list of check packages built into `fitness`. |
| `bundle-hashes` | Print archive hashes for installer build flags. |
| `assemble` | Gather GoReleaser outputs for native checks. |
| `smoke` | Test built assets without public downloads. |
| `verify-download` | Test published assets on the current host. |
| `record` | Print asset sizes and install time as JSON; see [performance record](../../../docs/ci.md#performance-record). |
| `verify-tag` | Match `GITHUB_REF_NAME` to the version file. |
| `promote` | Mark a verified release stable and latest without a downgrade. |
| `release-pr` | Apply the PR template to the release PR, linking each new [entry file](../fitness-check-changelog/README.md#entry-files) and the compare view at its head commit. |
| `release-notes` | Write the `GITHUB_REF_NAME` release description from the entry files added since the previous release tag. |

Invoke helpers with `go -C go run ./cmd/fitness-release COMMAND --root ..`. Pass `--host` to `bundle-hashes` or `assemble` when only this platform was built, as `tests/requirements.sh` does. A command warns about any flag it does not read. Later workflow jobs use the compiled verifier from `out` instead.

## Verification and recovery

`smoke` and `verify-download` exercise the installer and a compiled commit hook without Go on the consumer PATH. The workflow also runs the full suite through the public action. These gates run on Linux with Intel and ARM hosts; the darwin binaries are cross-compiled but not smoke-tested in CI.

Publication uses `.goreleaser-publish.yml` only while the Release Please release is a draft. Retries may replace draft assets; public releases skip publication and rerun checks. See [release recovery](../../../docs/ci.md#retry-a-release) for the workflow steps.

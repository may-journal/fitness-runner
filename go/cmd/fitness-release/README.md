---
relatedConfigurations: ['../../../.github/workflows/release.yml']
---

# fitness-release

Typed Go tooling for release packaging, publishing, and native checks. It replaces the shell release scripts. Only the build step needs Go installed; later jobs use this tool's compiled executable.

## Commands

From the repository root:

```text
go -C go run ./cmd/fitness-release build --root ..
go -C go run ./cmd/fitness-release tag --root ..
```

The build creates four runner bundles, four installers, and native verifier tools in `out`. The installers embed the bundle hashes. Verifier tools are CI artifacts, not files consumers must install.

## Verification

`smoke` tests the built assets. `verify-download` fetches the public assets and tests them. Both run the installer, action input handling, and a compiled Go commit hook with Go absent from the consumer PATH.

`verify-tag` checks the tag against the changelog heading. `publish` extracts release notes and invokes the GitHub CLI with the verified asset list. See the [distribution guide](../../../docs/distribution.md).


## Automation

`ensure-tag` pushes the changelog tag at the checked-out commit after main CI passes. It rejects a tag that points elsewhere and skips valid generated pin updates. The workflow supplies an App token so the tag starts the release job.

`promote` marks the verified release latest unless a newer version is already latest. `pin-release` opens the pin update PR and its tracking issue from current main. Both read the release version from `out/release.json`; the workflow runs them only after all public checks pass.

A `publish` retry checks existing asset bytes and uploads missing files without replacing published assets. A `pin-release` retry preserves the existing PR and review edits. See [automation setup and recovery](../../../docs/ci.md#release-automation) for token access and retry steps.

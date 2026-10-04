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

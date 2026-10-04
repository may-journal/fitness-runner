---
relatedConfigurations: ['../release-please-config.json', '../.goreleaser-bundles.yml', '../.goreleaser-tools.yml', '../.goreleaser-publish.yml']
---
<!-- cspell:ignore goreleaser -->


# Distribution

Consumers install source with `go install` or download verified static binaries from GitHub Releases. Each binary bundle contains the runner, checks, and changelog stamper.

## Versioning

Release Please owns semantic versions in `version.txt`, starting at `1.0.0`. Root tags such as `v1.0.0` identify binary releases. After verification, a `go/v1.0.0` alias points at the same commit for the module in `go/`. Go consumers use `@v1.0.0` or `@latest`; version 2 would require a module path migration to `/v2`.

Existing `go/v0.date.time` tags and their asset URLs remain valid. The installer and root action select legacy URLs for version 0 and root release tags for version 1. The root timestamp changelog records development changes and no longer sets release versions.

## Build and release gates

GoReleaser OSS 2.18.2 owns builds, archives, checksums, and publication. The [bundle config](../.goreleaser-bundles.yml) builds the runner and checks first. The [tools config](../.goreleaser-tools.yml) then builds installers with those bundle hashes embedded. Pull requests use snapshots and test all four platforms on native hosts without Go on the consumer PATH.

After main CI passes, Release Please maintains a release PR. Its merge creates a draft with notes and an explicit root version tag. The release workflow repeats native checks, then the [publish config](../.goreleaser-publish.yml) adds assets to that draft and publishes a prerelease. Public installer and action checks must pass before promotion to latest.

A verified release gains its Go module alias and a pin update PR through create-pull-request. Release Please ignores those `chore` pin commits for release decisions. See [CI setup](ci.md#release-automation) for App access, review, recovery, and consumer upgrades.

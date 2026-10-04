---
relatedConfigurations: ['../.fitnessrc.json']
---

# Distribution

Every channel delivers the same static binaries: the runner, the changelog stamper, and every check.

1. `go install` — compiles from source via the public Go module proxy. No artifacts involved, and every published version is cached immutably.
2. GitHub Releases — per-platform tarballs with a checksums file, built and uploaded by CI on every version tag.

## Versioning

The CHANGELOG timestamp is the only version, and release tags are derived from it. Heading `### 2026.07.19.0837` becomes tag `go/v0.20260719.837`. Major is pinned at 0, because Go reserves majors of 2 and up for `/vN` module paths. Minor is the date, patch is the minute, and ordering is preserved.

`go -C go run ./cmd/fitness-release tag --root ..` prints the tag for the newest heading; CI rejects a tag that does not match. Consumers reference the plain version (`@v0.20260719.837`) or `@latest`. The Go proxy caches every published version immutably.

## Release gate

[Binary distribution](../.github/workflows/distribution.yml) builds all four bundles and smoke-tests them on native hosts without Go on PATH. The release workflow publishes only after those checks pass. Each release includes a compiled `fitness-install` for each host, with its version and bundle hashes embedded. There is no shell launcher; CI downloads and invokes the native executable.

The [Go release tool](../go/cmd/fitness-release/README.md) owns packaging, publishing, and smoke checks. Green main CI tags its exact commit; the release workflow checks public downloads before marking the release latest. It then opens a PR to update the [root action](../action.yml) pin, installer, shared workflow refs, and setup examples. See [release automation](ci.md#release-automation) for access, review, retries, and consumer upgrades.

---
relatedConfigurations: ['../.fitnessrc.json']
---

# Distribution

Every channel delivers the same static binaries: the runner, the changelog stamper, and every check.

1. `go install` — compiles from source via the public Go module proxy. No artifacts involved, and every published version is cached immutably.
2. GitHub Releases — per-platform tarballs with a checksums file, built and uploaded by CI on every version tag.

## Versioning

The CHANGELOG timestamp is the only version, and release tags are derived from it. Heading `### 2026.07.19.0837` becomes tag `go/v0.20260719.837`. Major is pinned at 0, because Go reserves majors of 2 and up for `/vN` module paths. Minor is the date, patch is the minute, and ordering is preserved.

`.github/scripts/release-tag.sh` prints the tag for the newest heading, and CI refuses any tag that does not match. Consumers reference the plain version (`@v0.20260719.837`) or `@latest`. The Go proxy caches every published version immutably.

---
relatedConfigurations: ['../../.fitnessrc.json']
---

# 03 — Publish: releases, go install, brew

> The suite is done but landlocked — private repo, zero tags, no artifacts. Wire the three README distribution channels so consumers can install the binaries and pull new versions.

## Goal

`go install github.com/may-journal/fitness-runner/go/cmd/...@latest` works from any machine through the public Go module proxy. Release tags derive from the newest CHANGELOG heading — `### 2026.07.19.0837` becomes `go/v0.20260719.837` — so the stamper stays the only version authority. Every tag triggers a release workflow that cross-compiles static tarballs for darwin and linux on both architectures, writes a checksums file, and publishes a GitHub Release with the newest CHANGELOG section as its notes. `brew tap may-journal/tap && brew install fitness` installs the whole suite, and the release workflow bumps the formula so `brew upgrade` delivers each new version.

## Plan

0. Pre-flight

   - [x] Full-history audit for credentials, personal information, and stray blobs before the public flip
   - [x] Push local `main` to origin

1. Release automation

   - [x] `.github/workflows/release.yml` — on `go/v*` tag push: build all binaries per platform, tar per platform, checksums file, then publish the GitHub Release with the newest CHANGELOG section as notes
   - [x] `.github/scripts/release-tag.sh` derives the tag from the newest CHANGELOG heading; the workflow refuses a mismatched tag
   - [x] Flip the repo public
   - [x] Push the CHANGELOG-derived tag and confirm the workflow publishes the first release with all assets

2. Brew tap

   - [ ] Create public `may-journal/homebrew-tap` carrying `Formula/fitness.rb` — per-platform release tarball URLs with their checksums; installs every binary (deferred to issue #47)
   - [ ] The release workflow regenerates and pushes the formula on each tag (requires a `TAP_PUSH_TOKEN` repo secret; the job skips politely without it) (deferred to issue #47)

3. Verification

   - [x] `go install` of the tagged version resolves through the public Go proxy from a clean environment
   - [x] The release page shows four tarballs plus checksums, and an extracted binary runs
   - [ ] `brew install may-journal/tap/fitness` works on this machine and `fitness` runs (deferred to issue #47)

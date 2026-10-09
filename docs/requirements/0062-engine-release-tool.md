---
relatedConfigurations: ['../../.fitnessrc.json']
---

# 0062 Release Tool Reports Every Step

## Why

I build releases from `version.txt` and checked archives, so a wrong tag or changed file stops the release.

When a release step fails in CI, I see why in the job summary without digging through logs.

## Measurement

release commands that end with a clear outcome
-
release commands run

Source: `go/cmd/fitness-release/main.go:25`

## Requirements

- 0062.1
    - Given a repo whose `version.txt` says 1.0.1
        - When I run `version` and `tag`
            - Then they print 1.0.1 and v1.0.1 and exit 0
- 0062.2
    - Given a release tag that differs from `version.txt`
        - When `verify-tag` runs
            - Then it fails naming both versions and exits 1
- 0062.3
    - Given bundle archives that match their GoReleaser checksums
        - When `bundle-hashes` runs
            - Then it prints each platform with its hash
- 0062.4
    - Given build outputs that are missing or changed
        - When a build step reads them
            - Then it fails naming the file and exits 1
- 0062.5
    - Given no command, an unknown one, a bad flag, or an invalid version
        - When the release tool runs
            - Then it names the mistake and exits 1
- 0062.6
    - Given a release step inside a GitHub Actions job
        - When it passes or fails
            - Then the job summary shows its outcome and failures get an error annotation
- 0062.7
    - Given a release step outside GitHub Actions
        - When it fails
            - Then it writes no job summary
- 0062.8
    - Given a job summary path it cannot write
        - When a release step fails
            - Then it still prints the failure and exits 1

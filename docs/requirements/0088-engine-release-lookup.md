---
relatedConfigurations: ['../../.fitnessrc.json']
---

# 0088 Release Lookup Survives Rate Limits

## Why

My hooks find the newest Fitness release under my own GitHub limit, ask once per 10 minutes, and keep working when GitHub refuses.

## Measurement

latest release runs that install a version
-
latest release runs with a token, a recent answer, or an installed version

Source: `go/internal/distribution/latest.go:45`

## Requirements

- 0088.1
    - Given Developer has a GitHub token in `GH_TOKEN` or `GITHUB_TOKEN`
        - When I run `fitness-install --version latest`
            - Then the release lookup sends that token and never asks `gh` for one
- 0088.2
    - Given Developer is logged in to `gh` and sets no token variable
        - When I run `fitness-install --version latest`
            - Then the release lookup sends the token `gh auth token` prints
- 0088.3
    - Given Developer ran a release lookup less than 10 minutes ago
        - When I run `fitness-install --version latest` again
            - Then it installs the version it found without asking GitHub
- 0088.4
    - Given Developer has a recorded or installed version and GitHub refuses the lookup with HTTP 403
        - When I run `fitness-install --version latest`
            - Then it installs that version, prints one warning, and exits 0
- 0088.5
    - Given Developer has an exact release version, as the GitHub job passes
        - When I run `fitness-install --version` with it
            - Then it installs that release without a release lookup
- 0088.6
    - Given Developer has a stale recorded version and a releases API that never answers
        - When I run `fitness-install --version latest`
            - Then it names the host it asks, falls back to that version once five seconds pass, and exits 0

---
relatedConfigurations: ['../action.yml', '../go/cmd/fitness-install/main.go']
---

# Run prebuilt Fitness in CI

Fitness builds its installer, runner, and checks when publishing a release. Consumers download executables; they do not compile Fitness or install Go. Selected project checks may still need tools such as Go, Node, or SwiftLint.

The examples pin `go/v0.20261004.1140` and run all checks in external mode. The earlier external release requires a check list; this release makes it optional.

## Download the Go installer

This setup is for Linux amd64, from any directory. It installs into a private directory under your home and verifies the executable before running it. It needs curl and shasum, plus Git for file checks.

```bash
mkdir -p "$HOME/.local/fitness"
cd "$HOME/.local/fitness"
curl -fsSL https://github.com/may-journal/fitness-runner/releases/download/go/v0.20261004.1140/fitness-install-0.20261004.1140-linux-amd64 -o fitness-install-0.20261004.1140-linux-amd64
curl -fsSL https://github.com/may-journal/fitness-runner/releases/download/go/v0.20261004.1140/checksums.txt -o checksums.txt
grep '  fitness-install-0.20261004.1140-linux-amd64$' checksums.txt | shasum -a 256 -c -
chmod +x fitness-install-0.20261004.1140-linux-amd64
```

For another host, replace `linux-amd64` in the asset URL, filename, and later commands with the matching platform below. These are release assets, not paths to files checked into Git.

| Host | Platform |
| --- | --- |
| Linux, Intel or AMD 64-bit | `linux-amd64` |
| Linux, ARM 64-bit | `linux-arm64` |
| macOS, Intel | `darwin-amd64` |
| macOS, Apple silicon | `darwin-arm64` |

## Run checks

From the consumer repository root, run one command:

```bash
"$HOME/.local/fitness/fitness-install-0.20261004.1140-linux-amd64" -- --policy=external --all
```

The Go executable fetches a pinned bundle, verifies hashes and archive paths, and caches the binaries. Failed downloads, installs, or checks return a failing exit code. No copied shell installer is involved.

## Repo policy

To use external mode in both hooks and CI, commit this `.fitnessrc.json` in the consumer repo:

```json
{
  "policy": "external"
}
```

Then invoke the installer without runner flags to run every check. Each check decides whether it applies; applicable project checks may need their own tools. Use `--checks=name,name` or a config `checks` list only to request a subset. CLI selection replaces the config list; empty, unknown, or duplicate names fail.

A full-suite run includes new checks after an upgrade; an explicit list stays fixed and wins over `disabledChecks`. External mode never infers policy from the Git remote or installs hooks by default. Local file checks need no GitHub token or source upload; remote checks and project tools keep their own prerequisites. The runner preserves existing org behavior when policy is omitted.

## Jenkins

Save this complete `Jenkinsfile` in the consumer repo and configure a Pipeline from SCM job. Its `linux` agent must be Linux amd64 with curl and shasum. The temporary workspace directory holds the downloaded executable; the check runs in the consumer checkout.

```groovy
pipeline {
  agent { label 'linux' }
  options { skipDefaultCheckout(true) }
  stages {
    stage('Checkout') {
      steps { checkout scm }
    }
    stage('Download Fitness') {
      steps {
        dir("${env.WORKSPACE}@tmp/fitness") {
          sh '''
            curl -fsSL https://github.com/may-journal/fitness-runner/releases/download/go/v0.20261004.1140/fitness-install-0.20261004.1140-linux-amd64 -o fitness-install-0.20261004.1140-linux-amd64
            curl -fsSL https://github.com/may-journal/fitness-runner/releases/download/go/v0.20261004.1140/checksums.txt -o checksums.txt
            grep '  fitness-install-0.20261004.1140-linux-amd64$' checksums.txt | shasum -a 256 -c -
            chmod +x fitness-install-0.20261004.1140-linux-amd64
          '''
        }
      }
    }
    stage('Fitness') {
      steps {
        sh '"$WORKSPACE@tmp/fitness/fitness-install-0.20261004.1140-linux-amd64" -- --policy=external --all'
      }
    }
  }
}
```

The download stage only invokes standard tools. All installer logic runs in Go. Jenkins fails the check stage when Fitness returns a failure.

## GitHub Actions

Save as `.github/workflows/fitness.yml` in the consumer repo:

```yaml
name: Fitness
on: [push, pull_request]
permissions:
  contents: read
jobs:
  fitness:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: may-journal/fitness-runner@go/v0.20261004.1140
        with:
          policy: external
```

The action downloads the native installer and checks its hash, then invokes it. Go handles action inputs, outputs, caching, and check execution. Use a full action commit SHA if your team requires a fixed source ref.

## Commit hook

After downloading the installer, run this once from the consumer repository root:

```bash
"$HOME/.local/fitness/fitness-install-0.20261004.1140-linux-amd64" --install-hook -- --policy=external --all
```

This installs a compiled Go `pre-commit` executable and its check settings into Git's active hook directory. It refuses to replace existing hooks or their settings. No shell wrapper, Go toolchain, changelog stamp, or automatic file fix is involved.

The hook checks working-tree files, not an isolated staged snapshot. A failed check blocks the commit; CI runs independently of local hooks. Teams with an existing hook manager can invoke the installer through that manager instead.

## Cache and versions

`--install-only` prints the binary directory without running checks. Save that path and invoke its `fitness` executable for offline use. The compiled installer also works offline when its pinned bundle is cached.

The cache defaults to `$XDG_CACHE_HOME/fitness` or `$HOME/.cache/fitness`; `FITNESS_CACHE_DIR` chooses another private path. Go verifies archived and installed files before reuse. Repairs switch to a new copy without removing files used by running checks.

Upgrade or roll back by choosing an exact release with `--version VERSION`. `--version latest` is explicit opt-in; logs show the resolved version and hash. The action's `version` input follows the same rules.


## Release automation

A successful main CI run tags the exact commit using its changelog version. The release workflow builds and tests all four platforms, then checks the public downloads and action. Only a verified release becomes latest and opens a PR to update this repo's pins and setup examples. Pin PRs pass normal checks and review; their generated changes do not start another release.

The workflow uses `AUTOMATION_APP_ID` and `AUTOMATION_APP_KEY`. Install that GitHub App on this repo with write access to Contents, Pull requests, Issues, and Workflows. The App creates tags, a tracking issue, and the pin PR. Its token allows those events to start the next workflow and the PR checks.

## Retry a release

For a failed tag job, fix the cause and rerun the CI job. If a tag already exists at that commit, it is kept. A tag at a different commit is an error; add a new changelog version. 

For a failed release job, rerun the failed jobs in the Release workflow. Publish retries check existing asset bytes and upload missing files. They reject changed bytes. A pin retry keeps its existing PR and review edits.

## Consumer updates

This updates pins in the Fitness repo. A consumer on a fixed tag or commit SHA stays on that ref until its own update PR merges. Use an updater in each consumer repo to propose those changes. The explicit `latest` version option follows verified releases; it does not update the action's source ref.

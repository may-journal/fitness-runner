---
relatedConfigurations: ['../action.yml', '../go/cmd/fitness-install/main.go']
---

# Run prebuilt Fitness in CI

Fitness builds its installer, runner, and checks when publishing a release. Consumers download executables; they do not compile Fitness or install Go. Selected project checks may still need tools such as Go, Node, or SwiftLint.

The examples pin `v1.0.1` and run all checks in external mode. The earlier external release requires a check list; this release makes it optional.

## Download the Go installer

This setup is for Linux amd64, from any directory. It installs into a private directory under your home and verifies the executable before running it. It needs curl and shasum, plus Git for file checks.

```bash
mkdir -p "$HOME/.local/fitness"
cd "$HOME/.local/fitness"
curl -fsSL https://github.com/may-journal/fitness-runner/releases/download/v1.0.1/fitness-install-1.0.1-linux-amd64 -o fitness-install-1.0.1-linux-amd64
curl -fsSL https://github.com/may-journal/fitness-runner/releases/download/v1.0.1/checksums.txt -o checksums.txt
grep '  fitness-install-1.0.1-linux-amd64$' checksums.txt | shasum -a 256 -c -
chmod +x fitness-install-1.0.1-linux-amd64
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
"$HOME/.local/fitness/fitness-install-1.0.1-linux-amd64" -- --policy=external --all
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
            curl -fsSL https://github.com/may-journal/fitness-runner/releases/download/v1.0.1/fitness-install-1.0.1-linux-amd64 -o fitness-install-1.0.1-linux-amd64
            curl -fsSL https://github.com/may-journal/fitness-runner/releases/download/v1.0.1/checksums.txt -o checksums.txt
            grep '  fitness-install-1.0.1-linux-amd64$' checksums.txt | shasum -a 256 -c -
            chmod +x fitness-install-1.0.1-linux-amd64
          '''
        }
      }
    }
    stage('Fitness') {
      steps {
        sh '"$WORKSPACE@tmp/fitness/fitness-install-1.0.1-linux-amd64" -- --policy=external --all'
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
      - uses: may-journal/fitness-runner@v1.0.1
        with:
          policy: external
```

The action downloads the native installer and checks its hash, then invokes it. Go handles action inputs, outputs, caching, and check execution. Use a full action commit SHA if your team requires a fixed source ref.

## Commit hook

After downloading the installer, run this once from the consumer repository root:

```bash
"$HOME/.local/fitness/fitness-install-1.0.1-linux-amd64" --install-hook -- --policy=external --all
```

This installs a compiled Go `pre-commit` executable and its check settings into Git's active hook directory. It refuses to replace existing hooks or their settings. No shell wrapper, Go toolchain, changelog stamp, or automatic file fix is involved.

The hook checks working-tree files, not an isolated staged snapshot. A failed check blocks the commit; CI runs independently of local hooks. Teams with an existing hook manager can invoke the installer through that manager instead.

## Calling shared Go hooks

Hook managers can invoke the installer with `-- hook pre-commit` or `-- hook commit-msg` followed by the message file. The installer puts verified bundle tools on the child process PATH while preserving project tools. The shared pre-commit hook stamps the staged changelog; the external check-only hook above does not.

This behavior requires a release containing the bundled-tool PATH fix. Earlier installers need manual PATH setup for the shared pre-commit stamper.

## Cache and versions

`--install-only` prints the binary directory without running checks. Save that path and invoke its `fitness` executable for offline use. The compiled installer also works offline when its pinned bundle is cached.

The cache defaults to `$XDG_CACHE_HOME/fitness` or `$HOME/.cache/fitness`; `FITNESS_CACHE_DIR` chooses another private path. Go verifies archived and installed files before reuse. Repairs switch to a new copy without removing files used by running checks.

Upgrade or roll back by choosing an exact release with `--version VERSION`. `--version latest` is explicit opt-in; logs show the resolved version and hash. The action's `version` input follows the same rules. An exact action tag loads its matching installer; main, SHA, and local action refs use the verified fallback pin.

## Release automation

After main CI passes, Release Please groups conventional commits in a release PR and owns its version and GitHub notes. Once merged, it creates a draft release and an explicit root `v` tag. The root changelog remains the development audit; generated release notes stay on GitHub.

GoReleaser OSS 2.18.2 builds, archives, checksums, and publishes the assets after native checks on four platforms. It reuses the draft and publishes a prerelease for public download checks. Passing checks promote the release to latest and add its `go/v` module tag at the same commit. The create-pull-request action then proposes pin updates; `chore` pin commits do not request another release.

The workflow uses `AUTOMATION_APP_ID` and `AUTOMATION_APP_KEY`. Install that App with write access to Contents, Pull requests, Issues, and Workflows. Its token lets tags and PR events start workflows without a manual trigger. Release and pin PRs retain normal review, checks, and existing auto-merge rules.

## Retry a release

For a failed Release Please job, fix the cause and rerun the CI job. For a failed build or public check, rerun the failed jobs in the Release workflow. Each version keeps its existing tag and release.

GoReleaser can replace incomplete assets only while the release is a draft. Once public, the workflow skips publication and reruns verification. Pin PR updates use create-pull-request; review and merge remain in the normal PR flow.

## Consumer updates

This updates pins in the Fitness repo. A consumer on a fixed tag or commit SHA stays on that ref until its own update PR merges. Use an updater in each consumer repo to propose those changes. The explicit `latest` version option follows verified releases; it does not update the action's source ref.

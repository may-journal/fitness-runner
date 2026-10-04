---
relatedConfigurations: ['../action.yml', '../go/cmd/fitness-install/main.go']
---

# Run prebuilt Fitness in CI

Fitness builds its installer, runner, and checks when publishing a release. Consumers download executables; they do not compile Fitness or install Go. Selected project checks may still need tools such as Go, Node, or SwiftLint.

The examples pin `go/v0.20261004.1040` and select two local file checks in external mode. This release must be published and verified before these new examples are used. The prior release supports single checks but does not support external policy.

## Download the Go installer

This setup is for Linux amd64, from any directory. It installs into a private directory under your home and verifies the executable before running it. It needs curl and shasum, plus Git for file checks.

```bash
mkdir -p "$HOME/.local/fitness"
cd "$HOME/.local/fitness"
curl -fsSL https://github.com/may-journal/fitness-runner/releases/download/go/v0.20261004.1040/fitness-install-0.20261004.1040-linux-amd64 -o fitness-install-0.20261004.1040-linux-amd64
curl -fsSL https://github.com/may-journal/fitness-runner/releases/download/go/v0.20261004.1040/checksums.txt -o checksums.txt
grep '  fitness-install-0.20261004.1040-linux-amd64$' checksums.txt | shasum -a 256 -c -
chmod +x fitness-install-0.20261004.1040-linux-amd64
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
"$HOME/.local/fitness/fitness-install-0.20261004.1040-linux-amd64" -- --policy=external --checks=markdown-filename-kebab-case,markdown-links --all
```

The Go executable fetches a pinned bundle, verifies hashes and archive paths, and caches the binaries. Failed downloads, installs, or checks return a failing exit code. No copied shell installer is involved.

## Repo policy

To keep the same list in hooks and CI, commit this `.fitnessrc.json` in the consumer repo:

```json
{
  "policy": "external",
  "checks": ["markdown-filename-kebab-case", "markdown-links"]
}
```

Then invoke the installer without runner flags to use this list. CLI policy overrides config policy; CLI check selection replaces the whole config list. Empty, unknown, or duplicate names fail before checks run. An explicit list wins over `disabledChecks` and never grows on upgrade.

Without an explicit policy, Fitness keeps its org defaults and ignores a legacy `checks` list. External mode never infers policy from the Git remote or installs hooks by default. Local file checks need no GitHub token, org membership, or source upload; a first download still needs network access.

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
            curl -fsSL https://github.com/may-journal/fitness-runner/releases/download/go/v0.20261004.1040/fitness-install-0.20261004.1040-linux-amd64 -o fitness-install-0.20261004.1040-linux-amd64
            curl -fsSL https://github.com/may-journal/fitness-runner/releases/download/go/v0.20261004.1040/checksums.txt -o checksums.txt
            grep '  fitness-install-0.20261004.1040-linux-amd64$' checksums.txt | shasum -a 256 -c -
            chmod +x fitness-install-0.20261004.1040-linux-amd64
          '''
        }
      }
    }
    stage('Fitness') {
      steps {
        sh '"$WORKSPACE@tmp/fitness/fitness-install-0.20261004.1040-linux-amd64" -- --policy=external --checks=markdown-filename-kebab-case,markdown-links --all'
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
      - uses: may-journal/fitness-runner@go/v0.20261004.1040
        with:
          policy: external
          checks: markdown-filename-kebab-case,markdown-links
```

The action downloads the native installer and checks its hash, then invokes it. Go handles action inputs, outputs, caching, and check execution. Use a full action commit SHA if your team requires a fixed source ref.

## Commit hook

After downloading the installer, run this once from the consumer repository root:

```bash
"$HOME/.local/fitness/fitness-install-0.20261004.1040-linux-amd64" --install-hook -- --policy=external --checks=markdown-filename-kebab-case,markdown-links --all
```

This installs a compiled Go `pre-commit` executable and its check settings into Git's active hook directory. It refuses to replace existing hooks or their settings. No shell wrapper, Go toolchain, changelog stamp, or automatic file fix is involved.

The hook checks working-tree files, not an isolated staged snapshot. A failed check blocks the commit; CI runs independently of local hooks. Teams with an existing hook manager can invoke the installer through that manager instead.

## Cache and versions

`--install-only` prints the binary directory without running checks. Save that path and invoke its `fitness` executable for offline use. The compiled installer also works offline when its pinned bundle is cached.

The cache defaults to `$XDG_CACHE_HOME/fitness` or `$HOME/.cache/fitness`; `FITNESS_CACHE_DIR` chooses another private path. Go verifies archived and installed files before reuse. Repairs switch to a new copy without removing files used by running checks.

Upgrade or roll back by choosing an exact release with `--version VERSION`. `--version latest` is explicit opt-in; logs show the resolved version and hash. The action's `version` input follows the same rules.

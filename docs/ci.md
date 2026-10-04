---
relatedConfigurations: ['../action.yml', '../fitness.sh']
---

# Run prebuilt Fitness in CI

<!-- cspell:ignore Multibranch -->

Fitness builds its runner and checks when a release is published. Consumers download the binaries; they do not compile Fitness or install Go. Selected project checks can still need tools such as Go, Node, or SwiftLint.

The examples pin `go/v0.20261004.900` and run one check without org policy. Multiple chosen checks and external policy are tracked in issue #175. Leave out the runner arguments only when you want the current full suite.

## Shell

Use Bash, curl, tar, gzip, and either sha256sum or shasum on Linux or macOS, on amd64 or arm64. Run from a Git checkout. No GitHub token is needed for public release assets; the first download needs HTTPS access to GitHub.

```bash
bootstrap=$(curl -fsSL https://github.com/may-journal/fitness-runner/releases/download/go/v0.20261004.900/fitness.sh) && bash -c "$bootstrap" -- -- --check=markdown-filename-kebab-case --all
```

The first `--` names Bash's command; the second ends installer options. Downloads finish before the script runs. Failed downloads, installs, or checks fail the command.

## Jenkins

Save this as `Jenkinsfile` in the consumer repo. Create a Pipeline from SCM or Multibranch Pipeline job with that repo and its checkout credentials. The `linux` agent must have the shell tools listed above.

```groovy
pipeline {
  agent { label 'linux' }
  options { skipDefaultCheckout(true) }
  stages {
    stage('Checkout') {
      steps { checkout scm }
    }
    stage('Fitness') {
      steps {
        sh '''bootstrap=$(curl -fsSL https://github.com/may-journal/fitness-runner/releases/download/go/v0.20261004.900/fitness.sh) && bash -c "$bootstrap" -- -- --check=markdown-filename-kebab-case --all'''
      }
    }
  }
}
```

No local wrapper is required. Jenkins fails the stage if the command fails. Keep the repo's build and test stages alongside this stage.

## GitHub Actions

Save this as `.github/workflows/fitness.yml` in the consumer repo:

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
      - uses: may-journal/fitness-runner@go/v0.20261004.900
        with:
          check: markdown-filename-kebab-case
```

Use the same shell command from the Jenkins example in a `run` step if you prefer. The action uses the release pinned in its source and checks files in the consumer checkout. Use a full action commit SHA when your team requires a fixed source ref.

## Commit hook

From the consumer repo root, run this once per clone. It uses Git's current hook path and refuses to replace an existing hook. If a hook manager owns that path, add the Fitness command through that manager.

```bash
set -eu
hook=$(git rev-parse --path-format=absolute --git-path hooks/pre-commit)
if [ -e "$hook" ] || [ -L "$hook" ]; then
  echo "Existing hook: $hook; add Fitness through its owner." >&2
  exit 1
fi
mkdir -p "$(dirname "$hook")"
(set -C; cat > "$hook" <<'HOOK'
#!/usr/bin/env bash
set -euo pipefail
cd "$(git rev-parse --show-toplevel)"
bootstrap=$(curl -fsSL https://github.com/may-journal/fitness-runner/releases/download/go/v0.20261004.900/fitness.sh) && bash -c "$bootstrap" -- -- --check=markdown-filename-kebab-case --all
HOOK
)
chmod +x "$hook"
```

The hook checks working-tree files, not an isolated staged snapshot. It does not stamp changelogs or fix files. CI runs even when a local hook is bypassed.

## Offline use and cache

To install without running, pass `--install-only` to the published script. It prints the directory holding `fitness` and all check binaries. Save that path and invoke its `fitness` binary for offline use.

The cache defaults to `$XDG_CACHE_HOME/fitness` or `$HOME/.cache/fitness`. Set `FITNESS_CACHE_DIR` to choose another private path. Archives and installed binaries are checked against the release hash before reuse; damaged cache entries are replaced.

## Versions and rollback

The release URL pins both the installer and its bundle. To upgrade or roll back, change the URL or action ref to a tested release. `--version latest` and the action's `version: latest` are explicit opt-ins; logs show the resolved version and hash.

A source checkout of `fitness.sh` fetches the selected release's checksum list. The published script embeds its own release hashes, so it can run from a warm cache without network access. Fetching that script with curl still needs network access each time.

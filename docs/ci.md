---
relatedConfigurations: ['../action.yml', '../go/cmd/fitness-install/main.go']
---

# Run prebuilt Fitness in CI

Fitness builds its installer, runner, and checks when publishing a release. Consumers download executables; they do not compile Fitness or install Go. Selected project checks may still need the tools their language module runs.

The GitHub Actions example uses the `v1` major tag, which always points at the newest verified 1.x release. Download examples name `v1.2.0` and run all checks in external mode. [Install latest](#install-latest) follows the current published release.

## Download the Go installer

This setup is for Linux amd64, from any directory. It installs into a private directory under your home and verifies the executable before running it. It needs curl and shasum, plus Git for file checks.

```bash
mkdir -p "$HOME/.local/fitness"
cd "$HOME/.local/fitness"
curl -fsSL https://github.com/may-journal/fitness-runner/releases/download/v1.2.0/fitness-install-1.2.0-linux-amd64 -o fitness-install-1.2.0-linux-amd64
curl -fsSL https://github.com/may-journal/fitness-runner/releases/download/v1.2.0/checksums.txt -o checksums.txt
grep '  fitness-install-1.2.0-linux-amd64$' checksums.txt | shasum -a 256 -c -
chmod +x fitness-install-1.2.0-linux-amd64
```

For another host, replace `linux-amd64` in the asset URL, filename, and later commands with the matching platform below. These are release assets, not paths to files checked into Git.

| Host | Platform |
| --- | --- |
| Linux, Intel or AMD 64-bit | `linux-amd64` |
| Linux, ARM 64-bit | `linux-arm64` |
| macOS, Intel | `darwin-amd64` |
| macOS, Apple silicon | `darwin-arm64` |

## Install latest

This setup downloads the current published installer and verifies its checksum before installing it. It needs authenticated `gh` and `shasum`, with no Go compiler. Choose your platform from the table above; the example uses a Mac with Apple silicon.

The default destination is `$HOME/.local/share/fitness/fitness-install`. Set `FITNESS_INSTALL_DIR` before running the block to use another directory. The installer keeps a stable filename as releases change; the second command block runs it from the consumer repo root.

```sh
(
set -e
FITNESS_PLATFORM=darwin-arm64
FITNESS_RELEASE=$(gh release view --repo may-journal/fitness-runner --json tagName --jq .tagName)
FITNESS_INSTALL_DIR=${FITNESS_INSTALL_DIR:-"$HOME/.local/share/fitness"}
FITNESS_SETUP=$(mktemp -d)
cd "$FITNESS_SETUP"
gh release download "$FITNESS_RELEASE" --repo may-journal/fitness-runner --pattern "fitness-install-*-$FITNESS_PLATFORM" --pattern checksums.txt
grep "  fitness-install-.*-$FITNESS_PLATFORM$" checksums.txt | shasum -a 256 -c - && chmod +x fitness-install-*-"$FITNESS_PLATFORM" && mkdir -p "$FITNESS_INSTALL_DIR" && mv fitness-install-*-"$FITNESS_PLATFORM" "$FITNESS_INSTALL_DIR/fitness-install"
)
```

```sh
"${FITNESS_INSTALL_DIR:-$HOME/.local/share/fitness}/fitness-install" --version latest
```

Each invocation resolves the latest release online, verifies the bundle, and reuses its cache. Failed lookup or verification fails the command. Rerun setup to refresh the installer itself.

## Run checks

From the consumer repository root, run one command:

```bash
"$HOME/.local/fitness/fitness-install-1.2.0-linux-amd64" -- --policy=external --all
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

A full-suite run includes new checks after an upgrade; an explicit list stays fixed. External mode never infers policy from the Git remote or installs hooks by default. Local file checks need no GitHub token or source upload; remote checks and project tools keep their own prerequisites. The runner preserves existing org behavior when policy is omitted.

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
            curl -fsSL https://github.com/may-journal/fitness-runner/releases/download/v1.2.0/fitness-install-1.2.0-linux-amd64 -o fitness-install-1.2.0-linux-amd64
            curl -fsSL https://github.com/may-journal/fitness-runner/releases/download/v1.2.0/checksums.txt -o checksums.txt
            grep '  fitness-install-1.2.0-linux-amd64$' checksums.txt | shasum -a 256 -c -
            chmod +x fitness-install-1.2.0-linux-amd64
          '''
        }
      }
    }
    stage('Fitness') {
      steps {
        sh '"$WORKSPACE@tmp/fitness/fitness-install-1.2.0-linux-amd64" -- --policy=external --all'
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
      - uses: may-journal/fitness-runner@v1
        with:
          policy: external
```

The action downloads the native installer and checks its hash, then invokes it. Go handles action inputs, outputs, caching, and check execution. Use a full action commit SHA if your team requires a fixed source ref.

Findings fail the job with error annotations. A check that cannot run, such as one missing its tool, adds a `::notice` and a line in the job summary's Info section. Info never fails the job.

## Commit hook

After downloading the installer, run this once from the consumer repository root:

```bash
"$HOME/.local/fitness/fitness-install-1.2.0-linux-amd64" --install-hook -- --policy=external --all
```

This installs a compiled Go `pre-commit` executable and its check settings into Git's active hook directory. With no commit-msg hook, its `semantic-commit` judges the branch's earlier commits, so a bad subject fails the next commit. It refuses to replace existing hooks or their settings. No shell wrapper, Go toolchain, or automatic file fix is involved.

The hook checks working-tree files, not an isolated staged snapshot. A failed check blocks the commit; CI runs independently of local hooks. Teams with an existing hook manager can invoke the installer through that manager instead.

## Calling shared Go hooks

Hook managers can invoke the installer with `-- hook pre-commit` or `-- hook commit-msg` followed by the message file. The installer puts verified bundle tools on the child process PATH while preserving project tools.

## Cache and versions

`--install-only` prints the binary directory without running checks. Save that path and invoke its `fitness` executable for offline use. The compiled installer also works offline when its pinned bundle is cached.

The cache defaults to `$XDG_CACHE_HOME/fitness` or `$HOME/.cache/fitness`; `FITNESS_CACHE_DIR` chooses another private path. Go verifies archived and installed files before reuse. Repairs switch to a new copy without removing files used by running checks.

Upgrade or roll back with `--version VERSION`; `--version latest` is explicit opt-in. The action's `version` input follows the same rules. An exact action tag loads its installer; a major tag such as `v1` loads its newest release, and other refs load the latest.

## Network waits

Each network read first prints the host it asks and how long it waits. The latest release lookup and the hash manifest give up after five seconds. A failed lookup falls back to the recorded or newest installed version with one warning. The bundle download waits up to two minutes and prints its size before it starts, because it has no fallback.

The runner, `gh` and git reach the network through `HTTPS_PROXY`, `HTTP_PROXY` and `NO_PROXY`, and git also reads its `http.proxy` setting. A run with `local-branch`, `dependency-currency` beside a root `package.json`, or a commit message linking an Issue first probes the GitHub API host. The host is `GITHUB_API_URL` when set, else `api.github.com`, or `GH_HOST` for GitHub Enterprise Server. The probe connects once, through the proxy, within two seconds.

When the probe fails, those checks, `workflow-ref-exists` and the pre-push gate report info naming the host and proxy, without connecting. `issue-link-once` skips its open PR read and still judges the branch commits. Go checks then run with `GOPROXY=off`, so a missing module fails at once with Go's own message. A plain commit probes nothing, and when the probe passes, a slow registry or proxy still ends at the check's budget.

## Release automation

After main CI passes, Release Please groups conventional commits in a release PR and owns its version. Once merged, it creates a draft release and an explicit root `v` tag. `fitness-release release-pr` links each new [entry file](../go/cmd/fitness-check-changelog/README.md#entry-files) and the compare view at the PR's head commit, and commits nothing. While the release is a draft, `fitness-release release-notes` writes its description from the entry files added since the previous tag.

GoReleaser OSS 2.18.2 builds, archives, checksums, and publishes the assets after native checks on four platforms. It reuses the draft and publishes a prerelease for public download checks. Passing checks promote the release to latest and add its `go/v` module tag at the same commit. When that release is the newest, the workflow moves its major tag, such as `v1`, to the same commit.

The workflow uses `AUTOMATION_APP_ID` and `AUTOMATION_APP_KEY`. Install that App with write access to Contents, Pull requests, Issues, and Workflows. Its token lets tags and PR events start workflows without a manual trigger. Release PRs retain normal review, checks, and existing auto-merge rules.

## Retry a release

For a failed Release Please job, fix the cause and rerun the CI job. For a failed build or public check, rerun the failed jobs in the Release workflow. Each version keeps its existing tag and release.

GoReleaser can replace incomplete assets only while the release is a draft. Once public, the workflow skips publication and reruns verification.

## Performance record

`fitness --json=<file>` also writes the run as JSON: total `elapsedMs`, the runner's `runnerBytes`, and per check `ms`, `timeoutMs`, `timedOut`, `filesChecked`, `binaryBytes`, `errorLines`. A file it cannot write fails the run. In GitHub Actions, the job summary adds a check timing table, slowest first, with each check's share of its budget.

The shared CI writes this file and the Binary distribution smoke job adds `fitness-release record`. That command prints each release asset's size and the time to install from an empty cache. Release builds run the same job, so each release gets a record per Linux platform.

History is the trail of `fitness-performance-*` workflow artifacts, kept 90 days. Compare runs by downloading them with `gh run download`. A dashboard such as studio-pulse could read the same artifacts later, read-only; nothing in Fitness writes history elsewhere.

## Box diagram

`fitness boxes <report.json> <page.html>` draws a `--json` report as one HTML page. The shared CI writes it as `fitness.html` beside `fitness.json`, so each `fitness-performance-*` artifact holds both. To open it, download the artifact from the run's Summary page, or run `gh run download <run-id> --pattern 'fitness-performance-*'`. Then open `fitness.html` in any browser; it needs no network.

Each box is one check, and its area is that check's share of the chosen measure: runtime, binary size, or error lines. A check may print several lines for one problem, so the last counts lines. A green edge means passed and a red edge failed or timed out, and the box says so in words.

Checks with nothing to measure, such as no error lines, sit in a row of equal small boxes below. Hover or focus a box for its numbers, or open the table under the diagram for every value.

## Actions minutes

GitHub bills each job in a private repo rounded up to a whole minute, so job count drives the bill. `fitness-release minutes` totals it from the runs and jobs API:

```sh
go -C go run ./cmd/fitness-release minutes --org may-journal --days 4
```

It prints one Markdown row per workflow: jobs that ran, billed minutes, and the median job's seconds. A skipped job started no runner, so it counts neither. `--repos owner/a,owner/b` names repos instead of an org's private ones. GitHub lists at most 1,000 runs per repo for one window, so keep `--days` short on a busy repo.

## What stays

Three changes looked cheaper and were measured instead. None changes a whole-minute bill, so each stays as it is.

| Kept | Measured | Why it stays |
| --- | --- | --- |
| No cancel-in-progress on the suite | a cancelled 20 s suite job still bills one minute | cancelling a superseded push saves no minute |
| Full-history checkout | 1–3 s of a 20–60 s job | changelog-updated, semantic-commit and plan-trailer read the branch's commits |
| One process per check | a 62-check run took 1,681 ms, the same as its slowest check | the pool starts every check in parallel, so startup adds no measurable time |

## Consumer updates

Consumers pin the major tag, `@v1`, and get each verified 1.x release with no update PR. A new major version needs a one-line change to `@v2`.

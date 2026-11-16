# fitness-install

Installs a pinned release and runs Fitness without compiling consumer code. Release publishing embeds the version and the SHA-256 hashes of all four bundles. CI downloads and invokes the native executable directly.

## Usage

```bash
fitness-install -- markdown-filename-kebab-case --all
fitness-install --install-only
fitness-install --install-hook -- markdown-filename-kebab-case --all
fitness-install --version latest -- --help
```

Installer flags precede `--`; arguments after it reach Fitness unchanged. Install-only mode prints the directory holding the runner as its last line; with `--with-checks`, a line before it names the installed checks folder.

Built with `go install`, the installer defaults to the module version Go recorded, else latest. The runner replaces the installer process, so check exit codes and signals reach CI. `--install-hook` writes a compiled Go hook and saved options, preserving existing hooks.

## Latest release

`--version latest` asks the GitHub releases API for the newest release. The lookup sends `GH_TOKEN`, else `GITHUB_TOKEN`, else the token `gh auth token` prints. It then counts against that account's limit, not the one shared by every caller on an address. `GITHUB_API_URL` moves the API root, as GitHub jobs set it.

The answer is recorded in the cache as `latest-release.json`, and for 10 minutes later runs use it without a lookup. A failed lookup, such as HTTP 403 on a rate limit, prints one warning. The installer then uses the recorded answer or the newest installed version, and fails only with neither.

An exact `--version`, as the root action passes, never looks up.

## Checks

`--with-checks` also installs each check the repo pins in `checkVersions`, as `fitness check-versions` prints them. Each pin resolves through the fitness-checks release index, or through its tags while no index is published. A tag or index answer is reused for 10 minutes.

Each check archive must match its SHA-256 before its program is cached by release and platform. The installer hands the engine one folder of exactly the pinned checks as `FITNESS_CHECKS_DIR`. An install that fails offline fails the run, and a pin with no release warns and its check does not run. The action's `with-checks` input does the same and sets it for later steps.

Only `--install-only`, the action and a `--no-builtins` run install the checks up front. Any other run leaves them to the engine. It fetches only the checks the run's files reach, as the release index fields of each version say. A full run fetches every pinned check, and a reached check that cannot be fetched is an info row naming it.

## Cache

Downloads and archive paths are checked in Go. Each cached copy has a version, platform, and hash key. A repair publishes a new copy and atomically switches the current pointer; older copies remain available to running checks.

Set `FITNESS_CACHE_DIR` to choose a private cache. The default is `$XDG_CACHE_HOME/fitness` or `$HOME/.cache/fitness`. Delete an unused version's cache directory to reclaim its older copies.

## Tests

Go tests use a local HTTP server for download failures, unsafe archives, cache repair, and parallel installs. Native release tests run the compiled installer, a real check, a commit hook, and the root action with Go absent from PATH.

## Skills

`--link-skills <folder>` installs the release, links each bundled skill into the folder, and runs no checks. Repeat the flag for more folders. A missing, broken or older-bundle link is written again; a real folder or another link stays. See [development](../../../docs/development.md#skills).

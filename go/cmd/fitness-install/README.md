# fitness-install

Installs a pinned release and runs Fitness without compiling consumer code. Release publishing embeds the version and the SHA-256 hashes of all four bundles. CI downloads and invokes the native executable directly.

## Usage

```bash
fitness-install -- markdown-filename-kebab-case --all
fitness-install --install-only
fitness-install --install-hook -- markdown-filename-kebab-case --all
fitness-install --version latest -- --help
```

Installer flags precede `--`; arguments after it reach Fitness unchanged. Install-only mode prints the directory holding the runner and every check. The runner replaces the installer process, so check exit codes and signals reach CI. `--install-hook` writes a compiled Go hook and saved options, preserving existing hooks.

## Latest release

`--version latest` asks the GitHub releases API for the newest release. The lookup sends `GH_TOKEN`, else `GITHUB_TOKEN`, else the token `gh auth token` prints. It then counts against that account's limit, not the one shared by every caller on an address. `GITHUB_API_URL` moves the API root, as GitHub jobs set it.

The answer is recorded in the cache as `latest-release.json`, and for 10 minutes later runs use it without a lookup. A failed lookup, such as HTTP 403 on a rate limit, prints one warning. The installer then uses the recorded answer or the newest installed version, and fails only with neither.

An exact `--version`, as the root action passes, never looks up.

## Cache

Downloads and archive paths are checked in Go. Each cached copy has a version, platform, and hash key. A repair publishes a new copy and atomically switches the current pointer; older copies remain available to running checks.

Set `FITNESS_CACHE_DIR` to choose a private cache. The default is `$XDG_CACHE_HOME/fitness` or `$HOME/.cache/fitness`. Delete an unused version's cache directory to reclaim its older copies.

## Tests

Go tests use a local HTTP server for download failures, unsafe archives, cache repair, and parallel installs. Native release tests run the compiled installer, a real check, a commit hook, and the root action with Go absent from PATH.

## Skills

`--link-skills <folder>` installs the release, links each bundled skill into the folder, and runs no checks. Repeat the flag for more folders. A missing, broken or older-bundle link is written again; a real folder or another link stays. See [development](../../../docs/development.md#skills).

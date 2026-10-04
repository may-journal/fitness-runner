---
relatedConfigurations: ['../../../action.yml']
---

# fitness-install

Installs a pinned release and runs Fitness without compiling consumer code. Release publishing embeds the version and the SHA-256 hashes of all four bundles. CI downloads and invokes the native executable directly.

## Usage

```bash
fitness-install -- --check=markdown-filename-kebab-case --all
fitness-install --install-only
fitness-install --install-hook -- --check=markdown-filename-kebab-case --all
fitness-install --version latest -- --help
```

Installer flags precede `--`; arguments after it reach Fitness unchanged. Install-only mode prints the directory holding the runner and every check. The runner replaces the installer process, so check exit codes and signals reach CI. `--install-hook` writes a compiled Go hook and saved options, preserving existing hooks.

## Cache

Downloads and archive paths are checked in Go. Each cached copy has a version, platform, and hash key. A repair publishes a new copy and atomically switches the current pointer; older copies remain available to running checks.

Set `FITNESS_CACHE_DIR` to choose a private cache. The default is `$XDG_CACHE_HOME/fitness` or `$HOME/.cache/fitness`. Delete an unused version's cache directory to reclaim its older copies.

## Tests

Go tests use a local HTTP server for download failures, unsafe archives, cache repair, and parallel installs. Native release tests run the compiled installer, a real check, a commit hook, and the root action with Go absent from PATH.

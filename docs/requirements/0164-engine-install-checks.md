# 0164 Engine install checks

## Why

Each check ships as its own release. So the installer fetches each check a repo pins, verifies it, and hands the engine their folder.

## Measurement

Pinned checks the installer installs, verifies and hands the engine
-
Pinned checks

## Requirements

- 0164.1
    - Given Developer has a repo that pins one check to a release and one to a major
        - When I run `fitness-install --with-checks --install-only` twice
            - Then each pin resolves to its newest matching release, both install verified into one folder, and the second run downloads and asks nothing
- 0164.2
    - Given Developer has a repo that pins a check to a major with no release
        - When I run `fitness-install --with-checks`, with or without `--no-builtins`
            - Then it warns, naming the check and the pin, and that check does not run; naming it with `--no-builtins` fails
- 0164.3
    - Given Developer has a repo that pins one check
        - When I run that check through `fitness-install --with-checks`, with and without `--no-builtins`
            - Then the installed check judges the repo, printing no note that it replaces a built-in
- 0164.4
    - Given Developer has a repo that pins one check, and GitHub refuses the latest release lookup
        - When I run `fitness-install --with-checks --version latest`
            - Then it falls back to the recorded release, warns that no check installs, asks for none, and runs only the engine's own checks
- 0164.5
    - Given Developer has a repo that pins one uncached check, behind a proxy that never answers
        - When I run `fitness-install --with-checks`
            - Then it gives up within seconds after one unanswered index read and warns; a run within the minute asks nothing
- 0164.6
    - Given Developer has a repo pinning no check behind a silent proxy, and one pinning one, with `FITNESS_WITH_CHECKS=pinned` as hooks set it
        - When I run `fitness-install --install-only` in each
            - Then the first asks the network nothing, and the second installs its pinned check

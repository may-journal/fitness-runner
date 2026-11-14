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
            - Then each pin resolves to its newest matching release, both install verified into one folder, and the second run downloads nothing
- 0164.2
    - Given Developer has a repo that pins a check to a major with no release
        - When I run `fitness-install --with-checks`, with or without `--no-builtins`
            - Then it warns, naming the check and the pin, and the built-ins run; with `--no-builtins` it fails
- 0164.3
    - Given Developer has a repo that pins one check
        - When I run that check through `fitness-install --with-checks` with `--no-builtins`
            - Then the installed check judges the repo
- 0164.4
    - Given Developer has a repo that pins one check, and GitHub refuses the latest release lookup
        - When I run `fitness-install --with-checks --version latest`
            - Then it warns that no check installs, asks for none, and the built-ins run

---
relatedConfigurations: ['../../.fitnessrc.json']
---

# 0061 Installer Runs Verified Fitness

## Why

I run one installer anywhere and get the exact, verified Fitness release without compiling anything. When I misuse it, it tells me why.

## Measurement

installer runs that install a verified bundle or name the mistake
-
installer runs

## Requirements

- 0061.1
    - Given Developer has any folder
        - When I run `fitness-install --help`
            - Then it lists every installer option and exits 0
- 0061.2
    - Given Developer has options the installer cannot combine or read
        - When I run `fitness-install` with them
            - Then it says why and exits 1
- 0061.4
    - Given Developer has a cached install with a damaged check binary
        - When I run a check through `fitness-install` again
            - Then it installs a fresh verified copy and the check passes
- 0061.5
    - Given Developer has a folder that is not a git repo
        - When I run `fitness-install --install-hook`
            - Then it says it cannot find the hook folder and exits 1
- 0061.6
    - Given Developer has an unformatted Go file and Go on the `PATH`
        - When I run `fitness-install -- gofmt`
            - Then my own Go tools judge the file and it fails
- 0061.7
    - Given Developer has a GitHub job asking for install only
        - When I run `fitness-install --github-action`
            - Then it writes the folder to the job outputs and `PATH`, and summarizes success
- 0061.8
    - Given Developer has GitHub job inputs the installer cannot use
        - When I run `fitness-install --github-action`
            - Then it says why, summarizes the failure, and exits 1
- 0061.9
    - Given Developer has an exact release version with a `v` prefix
        - When I run `fitness-install --version` with it and `--install-only`
            - Then it prints the folder where each `fitness-check-<name>` runs its check

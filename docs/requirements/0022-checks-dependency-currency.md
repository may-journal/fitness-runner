---
relatedConfigurations: ['../../.fitnessrc.json']
---

# 0022 Checks Dependency Currency

## Why

I learn the moment a dependency I declare falls behind its latest release, so updates never pile up.

## Measurement

declared dependencies at their latest published version
-
declared dependencies

Source: `go/cmd/fitness-check-dependency-currency/main.go:227`

## Requirements

- 0022.1
    - Given every declared dependency installed at its latest version
        - When dependency-currency runs
            - Then it passes
- 0022.2
    - Given a dependency, dev dependency, and peer dependency behind latest
        - When dependency-currency runs
            - Then it fails listing each as `name: current → latest`
- 0022.3
    - Given a declared dependency that is not installed
        - When dependency-currency runs
            - Then it fails listing it as `name: missing → latest`
- 0022.4
    - Given an internal `@mayjournal` package that is not installed
        - When dependency-currency runs
            - Then it passes, since internal packages are skipped
- 0022.5
    - Given a workspace dependency behind latest, in either `workspaces` form
        - When dependency-currency runs
            - Then it fails listing that dependency
- 0022.6
    - Given workspaces sharing one outdated install and one with its own install
        - When dependency-currency runs
            - Then it lists each installed version once
- 0022.7
    - Given an unreachable registry in the repo or home `.npmrc`
        - When dependency-currency runs
            - Then it passes rather than blocking the commit
- 0022.8
    - Given a repo without a `package.json`
        - When dependency-currency runs
            - Then it passes with no files checked

---
relatedConfigurations: ['../../.fitnessrc.json']
---

# 0047 Checks Node Version

## Why

I learn at once when my Node is older than the repo's `.nvmrc` asks, before a build fails strangely.

## Measurement

JavaScript repos whose Node satisfies `.nvmrc`
-
JavaScript repos checked

Source: `go/cmd/fitness-check-node-version/main.go:50`

## Requirements

- 0047.1
    - Given a repo with no `package.json`
        - When fitness runs node-version
            - Then it passes having checked no files
- 0047.2
    - Given a JavaScript repo without an `.nvmrc`
        - When fitness runs node-version
            - Then it fails with `missing .nvmrc`
- 0047.3
    - Given an `.nvmrc` major at or below the installed Node
        - When fitness runs node-version
            - Then it passes
- 0047.4
    - Given an `.nvmrc` with a `v` prefix
        - When fitness runs node-version
            - Then it reads the version the same way
- 0047.5
    - Given an `.nvmrc` major above the installed Node
        - When fitness runs node-version
            - Then it fails naming the required major and `nvm use`
- 0047.6
    - Given an `.nvmrc` with no version number
        - When fitness runs node-version
            - Then it fails reporting the requirement as `NaN.x`
- 0047.7
    - Given no `node` on the `PATH`
        - When fitness runs node-version
            - Then it fails telling me to install Node

---
relatedConfigurations: ['../../.fitnessrc.json']
---

# 0094 Dependency approve

## Why

A green pull request that only bumps dev dependencies by a patch or minor version merges without my approving it by hand. A runtime dependency, a major bump, or any other file still waits for me.

## Measurement

Green dev-dependency bump pull requests the approver App approved
-
Green dev-dependency bump pull requests

Source: #524

## Requirements

- 0094.1
    - Given Developer has a `build(deps)` pull request bumping a dev dependency's minor version, then one changing only the lockfile
        - When Release approve checks each as a dependency bump
            - Then it passes both on to the green-checks wait at their head commit
- 0094.2
    - Given Developer has a `build(deps)` pull request changing a runtime dependency, then one moving a runtime package in the lockfile
        - When Release approve checks each as a dependency bump
            - Then it approves neither and names what changed in a notice
- 0094.3
    - Given Developer has a `build(deps)` pull request that bumps a dev dependency by a major version
        - When Release approve checks it as a dependency bump
            - Then it approves nothing and names the package and both versions in a notice
- 0094.4
    - Given Developer has a pull request not titled `build(deps)`, then a `build(deps)` one that also changes a workflow
        - When Release approve checks each as a dependency bump
            - Then it approves neither, reads no file contents, and says why in a notice
- 0094.5
    - Given Developer has a caller passing kind `dependency`, then one passing no kind
        - When Release approve runs
            - Then the first runs only the dependency check, and the second only the release check

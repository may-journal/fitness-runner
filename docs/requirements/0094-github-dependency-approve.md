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
    - Given Developer has a `build(deps)` bump of a dev dependency's minor version, a lockfile-only one, then one with a legacy lockfile tree
        - When Release approve checks each against its merge base
            - Then it passes each on to the green-checks wait at its head commit
- 0094.2
    - Given Developer has a `build(deps)` bump changing a runtime dependency, a runtime package, a download URL, or a package in place
        - When Release approve checks each as a dependency bump
            - Then it approves none and names what changed in a notice
- 0094.3
    - Given Developer has a `build(deps)` pull request bumping a dev dependency's major version, then a 0.x minor version
        - When Release approve checks it as a dependency bump
            - Then it approves neither and names the package and both versions in a notice
- 0094.4
    - Given Developer has a pull request not titled `build(deps)`, then a `build(deps)` one that also changes a workflow
        - When Release approve checks each as a dependency bump
            - Then it approves neither, reads no file contents, and says why in a notice
- 0094.5
    - Given Developer has a caller passing kind `dependency`, one passing no kind, then one misspelling it
        - When Release approve runs
            - Then the first runs only the dependency check, the second only the release check, and the third fails naming the kind

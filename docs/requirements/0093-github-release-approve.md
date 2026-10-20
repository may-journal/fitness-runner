---
relatedConfigurations: ['../../.fitnessrc.json']
---

# 0093 Release approve

## Why

A green Release Please pull request merges without my approving it by hand. Any pull request that changes more than the release files, or is not green, still waits for me.

## Measurement

Green release pull requests the approver App approved
-
Green release pull requests

Source: #523

## Requirements

- 0093.1
    - Given Developer has a release pull request by the automation App that changes only `CHANGELOG.md`, `version.txt` and the manifest, with green checks
        - When Release approve runs on it twice
            - Then the first run approves it at the commit it checked, and the second approves nothing new
- 0093.2
    - Given Developer has a release pull request that also changes a workflow, or renames one into a release file
        - When Release approve runs on it
            - Then it approves nothing, reads no checks, and names the other file in a notice
- 0093.3
    - Given Developer has a pull request by someone other than the automation App
        - When Release approve runs on it
            - Then it approves nothing and names the author in a notice
- 0093.4
    - Given Developer has a release pull request with a failed check, then one whose checks never finish
        - When Release approve runs on each
            - Then it approves neither, with a notice for the failed check and a warning for the unfinished ones
- 0093.5
    - Given Developer has a release pull request whose only running check is Release approve's own, then one pushed to while its checks ran
        - When Release approve runs on each
            - Then it approves the first and not the second
- 0093.6
    - Given Developer has a repo without the approver App's secrets
        - When Release approve runs
            - Then it passes with a notice, and every later step but the report skips

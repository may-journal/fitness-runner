---
relatedConfigurations: ['../../.fitnessrc.json']
---

# 0086 Checks Workflow Ref Exists

## Why

No workflow I merge calls a may-journal path missing at its tag or sha. So no repo fails every event for a missing workflow.

## Measurement

may-journal `uses:` lines whose path exists at their tag or sha
-
may-journal `uses:` lines pinned to a tag or sha

Source: `go/cmd/fitness-check-workflow-ref-exists/main.go:334`

## Requirements

- 0086.1
    - Given Developer has workflows calling a may-journal workflow at a tag and an action at a sha, both present there
        - When workflow-ref-exists runs
            - Then it passes
- 0086.2
    - Given Developer has a workflow calling a may-journal workflow missing at its tag
        - When workflow-ref-exists runs
            - Then it fails naming the file, line, path, and tag
- 0086.3
    - Given Developer has workflows calling a may-journal action at a ref name and at a sha, neither of which exists
        - When workflow-ref-exists runs
            - Then it fails naming the file, line, and ref of each
- 0086.4
    - Given Developer has workflows calling local actions, other owners, expression refs, a may-journal branch, and a run script that writes a `uses:` line
        - When workflow-ref-exists runs
            - Then it passes and reads no paths from GitHub
- 0086.5
    - Given Developer has a workflow calling a may-journal repo the token cannot read
        - When workflow-ref-exists runs
            - Then it passes with a note naming that repo
- 0086.6
    - Given Developer has a repo with no workflows
        - When workflow-ref-exists runs
            - Then it passes having checked no files
- 0086.7
    - Given Developer has a workflow with a missing path, then a call GitHub refuses, then a call GitHub cannot reach
        - When workflow-ref-exists runs
            - Then it fails on the missing path and notes each call it could not verify

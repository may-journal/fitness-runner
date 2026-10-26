# 0067 Local Branch Contexts

## Why

My push checks its PR and Plans before Actions, while commits remain available offline.

## Measurement

entry points that run metadata only in their intended context
-
entry points exercised

## Requirements

- 0067.1
    - Given Developer has an invalid PR and an explicit local-branch selection
        - When I run the selected checks
            - Then one metadata execution reports findings in the shared results table
- 0067.2
    - Given Developer has the installed local-branch binary
        - When I request its describe metadata
            - Then it names the check and declares its timeout without GitHub reads
- 0067.3
    - Given Developer has an approved feature Plan and an invalid linked PR, or a PR read outlasting the push's remaining deadline
        - When I push with the shared hook
            - Then local-branch runs once and refuses the push
- 0067.5
    - Given Developer has the shared hooks and a GitHub branch
        - When I commit a valid change
            - Then the commit succeeds without local-branch or GitHub reads, apart from the installer's release lookup
- 0067.6
    - Given Developer has an Actions run in a GitHub checkout
        - When I run the full Fitness suite
            - Then every installed check passes and local-branch makes no GitHub reads
- 0067.7
    - Given Developer has a repo without a GitHub origin
        - When I run local-branch
            - Then it passes without targets or GitHub reads
- 0067.8
    - Given Developer has a fork PR whose API base commit is missing locally
        - When I run local-branch
            - Then it fails with fetch guidance naming the base repository
- 0067.9
    - Given Developer has an approved feature Plan and a valid linked PR
        - When I push with the shared hook
            - Then local-branch runs once on the hook's read of that Plan, and the push succeeds

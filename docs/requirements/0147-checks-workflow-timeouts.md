# 0147 Checks Workflow Timeouts

## Why

Every long workflow timeout I merge says why it is that long. So a slow regression cannot hide under a generous limit.

## Measurement

workflow timeouts of 10 minutes or more that cite an observed maximum and date
-
workflow timeouts of 10 minutes or more

## Requirements

- 0147.1
    - Given Developer has a workflow job whose `timeout-minutes` of 10 or more has no comment citing its observed maximum
        - When workflow-timeouts runs
            - Then it fails naming the file, the line and the comment to add
- 0147.2
    - Given Developer has a workflow job whose long timeout cites its observed maximum and date, and a short one without a comment
        - When workflow-timeouts runs
            - Then it passes

# 0133 Checks Workflow Reporting

## Why

Every workflow job I merge reports on its run page, even when it fails. So no failure shows only a red badge.

## Measurement

workflow jobs with steps that have an `always()` step writing the job summary
-
workflow jobs with steps

## Requirements

- 0133.1
    - Given Developer has a job whose last step runs with `always()` and writes the job summary
        - When workflow-reporting runs
            - Then it passes
- 0133.2
    - Given Developer has a `.yaml` workflow whose job has steps but none reports
        - When workflow-reporting runs
            - Then it fails naming the file, the job's line, and the step to add
- 0133.3
    - Given Developer has a job whose summary step runs only when earlier steps pass
        - When workflow-reporting runs
            - Then it fails naming that step's line
- 0133.4
    - Given Developer has a job that only calls a reusable workflow
        - When workflow-reporting runs
            - Then it passes
- 0133.5
    - Given Developer has a matrix job with no reporting step
        - When workflow-reporting runs
            - Then it fails naming that job once
- 0133.6
    - Given Developer has a workflow that is not valid YAML
        - When workflow-reporting runs
            - Then it fails naming the file and the line
- 0133.7
    - Given Developer has a repo with no workflows
        - When workflow-reporting runs
            - Then it passes having checked no files
- 0133.8
    - Given Developer has the workflows fitness-runner ships
        - When workflow-reporting runs
            - Then it passes

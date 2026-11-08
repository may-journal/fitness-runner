# 0003 — Every workflow provides native GitHub feedback

Every workflow job reports its outcome natively.

https://github.com/may-journal/fitness-runner/issues/855

## Check

- Reads each workflow in `.github/workflows`.
- Fails a workflow whose jobs with steps outnumber its lines that write `GITHUB_STEP_SUMMARY`.
- Fix: give the job a step that writes its outcome to the job summary, as `ci-reusable.yml` does.

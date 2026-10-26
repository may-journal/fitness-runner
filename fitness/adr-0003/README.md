# ADR 0003 check

Keeps [ADR 0003](../../docs/architecture/adr/0003-native-github-workflow-feedback.md): every workflow job reports its outcome natively.

- Reads each workflow in `.github/workflows`.
- Fails a workflow whose jobs with steps outnumber its lines that write `GITHUB_STEP_SUMMARY`.
- Fix: give the job a step that writes its outcome to the job summary, as `ci-reusable.yml` does.

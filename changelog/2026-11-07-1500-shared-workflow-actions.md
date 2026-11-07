### 2026.11.07.1500

- Perf: a newer push or edit to a pull request cancels the Fitness suite run it supersedes, so only the newest bills.
- Refactor: the workflows share their install, run, report and App token steps as composite actions under `.github/actions`, which reusable workflows name with `$/`.
- Ci: this repo's issue, comment and close checks call the reusable workflows they publish, built from their own commit.
- Ci: each job's always-run report step is one line, and only jobs that run Fitness upload its reports and performance record.

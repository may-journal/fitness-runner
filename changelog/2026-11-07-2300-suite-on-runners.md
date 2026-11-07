### 2026.11.07.2300

- Perf: the Fitness suite on a private repository's pull requests runs on the self-hosted `fitness` runners, so it bills no Actions minutes.
- Perf: pr-check-reusable and dupe-check-reusable pick those runners too, and a public repository or another repository's pull request stays hosted.
- Fix: on macOS the suite installs SwiftLint's official macOS build and keeps its Go, npm and SwiftLint caches on the runner's disk.
- Fix: every SwiftLint download must match the SHA-256 GitHub publishes for that release asset, so a changed zip fails the job.
- Docs: the workflows README says where the suite runs and how it caches on a Mac.

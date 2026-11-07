### 2026.11.07.1900

- Perf: comment, issue and close checks in private repos run on the self-hosted `fitness` runners, so they bill no Actions minutes.
- Ci: public repos keep those checks on `ubuntu-24.04-arm`, so a fork's pull request never reaches a self-hosted runner.
- Docs: the workflows README says where each of those checks runs.
- Fix: they pick the runners by the `fitness` group, and a fork's pull request stays hosted even in a private repo.

### 2026.10.20.1903

- Fix: Release approve fails before its 20-minute wait when `APPROVER_TOKEN` is rejected or cannot see the repo.
- Fix: the approver's login is read once, with its own token, in the token check.
- Test: requirement 0093 checks which token each step uses, and a replayed `gh` call can require the token it runs with.

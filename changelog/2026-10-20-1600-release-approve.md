### 2026.10.20.1600

- Feat: Release approve lets a second App approve a green Release Please pull request, so it merges without a person.
- Feat: it approves only an automation pull request changing `CHANGELOG.md`, `version.txt` and the manifest, and leaves others for a person.
- Feat: Release Please runs call it for an open release pull request, and it waits for every check but its own run.
- Docs: the workflows README explains when it approves and what the approver App needs.
- Test: requirement 0093 covers the approval, each refusal, the wait and missing secrets.

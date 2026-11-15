### 2026.11.15.1702

> A release without built-in checks passes its download check, so it publishes and `v1` moves to it.

- Fix: the release check runs the released action with `with-checks: pinned`, so it installs the checks this repo pins.
- Fix: 1.69.0 stayed a prerelease because that check found no check binary to run.
- Chore: `v1` stays on 1.68.0 until a release passes the check.

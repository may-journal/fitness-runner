### 2026.11.14.1140

- Test: requirement 0067 builds `local-branch` without the engine and expects exit 3.
- Test: the requirement builds the check in an outside module, since the installed binary carries the engine.
- Fix: the requirements jobs for shard 2 pass again on linux-amd64 and linux-arm64.

# 0002 — One binary per check retires the shared-lint-path decision

One lint path, owned by the eslint check.

https://github.com/may-journal/fitness-runner/issues/854

## Check

- Reads every tracked Go, JSON, YAML, script and hook file.
- Fails one that names the shared package's lint command, which was the retired second path.
- Fix: lint through the eslint check and its one config.

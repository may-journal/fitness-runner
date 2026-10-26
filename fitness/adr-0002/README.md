# ADR 0002 check

Keeps [ADR 0002](../../docs/architecture/adr/0002-one-binary-per-check-no-shared-lint-path.md): one lint path, owned by the eslint check.

- Reads every tracked Go, JSON, YAML, script and hook file.
- Fails one that names the shared package's lint command, which was the retired second path.
- Fix: lint through the eslint check and its one config.

# ADR 0010 check

Keeps [ADR 0010](../../docs/architecture/adr/0010-one-binary-holds-every-check.md): one `fitness` binary holds every built-in check.

- Fails a `go/cmd/fitness-check-<name>` package that exports no `Main`.
- Fails `go/cmd/fitness/checks.go` when it is no longer generated.
- Fix: export `Main` from the check package, and run `go -C go generate ./cmd/fitness`.

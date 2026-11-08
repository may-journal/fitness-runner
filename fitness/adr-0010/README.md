# 0010 — One binary holds every check

One `fitness` binary holds every built-in check.

https://github.com/may-journal/fitness-runner/issues/862

## Check

- Fails a `go/cmd/fitness-check-<name>` package that exports no `Main`.
- Fails `go/cmd/fitness/checks.go` when it is no longer generated.
- Fix: export `Main` from the check package, and run `go -C go generate ./cmd/fitness`.

# ADR 0012 check

Keeps [ADR 0012](../../docs/architecture/adr/0012-repo-checks-live-in-the-fitness-folder.md): fitness keeps its own checks in its `fitness/` folder. It is written in Go on the [check SDK](../../go/check/check.go), so it proves the Go path too.

- Fails `.fitnessrc.json` when it names a `localChecks` folder.
- Fails a tracked `fitness-check-<name>` program outside `go/cmd`.
- Fix: move the check into `fitness/<name>/` with a README.md.

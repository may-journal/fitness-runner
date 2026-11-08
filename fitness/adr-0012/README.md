# 0012 — A repo's own checks live in its fitness folder

Fitness keeps its own checks in its `fitness/` folder.

https://github.com/may-journal/fitness-runner/issues/864

## Check

- Fails `.fitnessrc.json` when it names a `localChecks` folder.
- Fails a tracked `fitness-check-<name>` program outside `go/cmd`.
- Fix: move the check into `fitness/<name>/` with a README.md.

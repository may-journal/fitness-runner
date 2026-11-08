# 0001 — Fix the work, not the limit: two totals guard the run's time

Two totals guard the run's time, and a slow check gets its work cut.

https://github.com/may-journal/fitness-runner/issues/853

## Check

- Fails a built-in check in the test stage with no entry in [config.json](config.json) naming the run it waits on.
- Fails an entry in `config.json` whose check is no longer in the test stage.
- Fails a `timeoutMs` key in `.fitnessrc.json`, a limit per check the run no longer has.
- Fix: cut the work so the check stays fast, or add its entry with the test run it waits on.

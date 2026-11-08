# Self-hosted runners

Comment, issue and close checks run on two runner processes on the agents' Mac, so they bill no Actions minutes. So does the suite on a private repo's pull requests. Research and risks live in [the research issue](https://github.com/may-journal/fitness-runner/issues/819).

## Set up

```sh
sudo scripts/fitness-runners.sh
```

It creates the `fitness` runner group for private repos only. It then creates the `fitness-runner` macOS user, with none of the agents' `gh` login or keychain. Next it registers `vanderbot-fitness-1` and `vanderbot-fitness-2` in the group. Each starts as a launchd job in `/Library/LaunchDaemons` that restarts it on exit.

Your `gh` login needs the org's runners admin permission. The registration token is used once and never saved.

`--dry-run --out <dir>` writes only the two launchd jobs and changes nothing else. `--org`, `--user` and `--count` pick another org, user and number of runners.

## Where checks run

The reusable workflows pick the `fitness` group in a private repo and `ubuntu-24.04-arm` in a public one. A fork's pull request stays hosted even in a private repo, so its code never reaches the Mac.

## Keep it running

- Keep the Mac awake in its energy settings. A job waits 24 hours for a runner, then fails.
- The Watchdog must leave the `com.vanderbot.fitness-runner.*` jobs alone.
- Each runner writes `launchd.log` in its folder under `/Users/fitness-runner`.
- Print the billed minutes with `go -C go run ./cmd/fitness-release minutes -org may-journal -days 7`; the [CI guide](ci.md#actions-minutes) explains the table.

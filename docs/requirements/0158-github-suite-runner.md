# 0158 Suite runner

## Why

My repo's checks that build Apple apps run on a hosted Mac, even in a public repo. No variable can send the suite to a self-hosted or billed larger runner.

## Measurement

Repos whose suite runs on the runner their checks need

-

Repos that run the suite

## Requirements

- 0158.1
    - Given Developer sets the repo's `FITNESS_RUNS_ON` variable to a standard hosted macOS label such as `xcode-27` or `macos-26`
        - When the Fitness suite picks its runner
            - Then it runs on that label
- 0158.2
    - Given Developer leaves `FITNESS_RUNS_ON` unset, or sets it to a self-hosted, Linux or `-large` label
        - When the Fitness suite picks its runner
            - Then it keeps the usual runner choice

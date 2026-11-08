# 0157 Self-hosted runners come from one script

## Why

I set up the Fitness runners with one checked-in script, so a new Mac matches the old one.

## Measurement

runner setups that write a launchd job for every runner as its own user

-

runner setups

## Requirements

- 0157.1
  - Given Developer has the setup script and asks for three runners as `ci-user`
    - When I run `scripts/fitness-runners.sh --dry-run`
      - Then it writes one keep-alive launchd job per runner, each run as `ci-user`, and changes nothing else
- 0157.2
  - Given Developer has the setup script
    - When I ask for zero runners
      - Then it names the bad count and exits 1

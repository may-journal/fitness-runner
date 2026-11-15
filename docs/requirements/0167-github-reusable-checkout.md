# 0167 GitHub Reusable Checkout

## Why

Each fitness command installs the checks a repo pins in its `.fitnessrc.json`. A job without a checkout has no config, so it installs no check and its command judges nothing.

## Measurement

Reusable workflow jobs that run a fitness command after a checkout
-
Reusable workflow jobs that run a fitness command

## Requirements

- 0167.1
    - Given Developer has the reusable workflows
        - When I read each job that runs the fitness-command action
            - Then a checkout step comes before it, and a failure names the workflow and the job

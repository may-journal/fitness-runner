# 0010 Judge Each Package

## Why

My monorepo names its package folders once, so every check judges each package on its own.

## Measurement

checks that get the package roots the repo's settings name

-

checks

## Requirements

- 0010.1
  - Given Developer has a monorepo whose `packages` setting names its package folders by glob, or its root as `.`
    - When the suite runs
      - Then each check gets the matched roots, a glob matching no folder is red naming it, and no setting keeps the repo root

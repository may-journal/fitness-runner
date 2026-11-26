# 0010 Judge Each Package

## Why

My monorepo names its package folders once, so every check judges each package on its own.

## Measurement

checks that get the package roots the repo's settings name

-

checks

## Requirements

1. Gives each check the package roots
   1. GIVEN Developer has a monorepo whose `packages` setting names its package folders by glob or its root as `.`
      1. WHEN the suite runs
         1. THEN each check gets the matched roots
         2. THEN a glob matching no folder is red naming it
         3. THEN no setting keeps the repo root

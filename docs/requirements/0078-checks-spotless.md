---
relatedConfigurations: ['../../.fitnessrc.json']
---

# 0078 Checks Spotless

## Why

I keep Java formatted the way my build says. The build's own tools check it in one run, with no daemon left behind.

## Measurement

Java repos whose Spotless check finds nothing to rewrite reported green
-
Java repos checked

## Requirements

- 0078.1
    - Given Developer has a build where Spotless finds unformatted files
        - When fitness runs spotless
            - Then it fails naming each file and the apply task
- 0078.2
    - Given Developer has a multi-module build with formatted sources
        - When fitness runs spotless
            - Then it passes after one run from the build root
- 0078.3
    - Given Developer has a build wrapper in the repo and the build tool on the `PATH`
        - When fitness runs spotless
            - Then it runs the wrapper
- 0078.4
    - Given Developer has a Gradle build
        - When fitness runs spotless
            - Then Gradle runs without a daemon
- 0078.5
    - Given Developer has a build that does not declare Spotless
        - When fitness runs spotless
            - Then it fails with the plugin snippet for that build
- 0078.6
    - Given Developer has a repo with no Maven or Gradle build
        - When fitness runs spotless
            - Then it passes with zero files
- 0078.7
    - Given Developer has a Gradle build that fails before the plugin runs
        - When fitness runs spotless
            - Then it shows each cause line Gradle gives

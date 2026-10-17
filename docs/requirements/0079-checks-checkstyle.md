---
relatedConfigurations: ['../../.fitnessrc.json']
---

# 0079 Checks Checkstyle

## Why

I keep Java clean and simple, because every Checkstyle finding, even a warning, turns the run red, and my config must cap complexity.

## Measurement

Java repos with a clean Checkstyle run and a complexity ceiling reported green
-
Java repos checked

Source: `go/internal/lang/java/checkstyle/checkstyle.go:120`

## Requirements

- 0079.1
    - Given Developer has a source file with a Checkstyle warning
        - When fitness runs checkstyle
            - Then it fails naming the file, line, column, message and rule
- 0079.2
    - Given Developer has clean sources and a config with a complexity ceiling
        - When fitness runs checkstyle
            - Then it passes
- 0079.3
    - Given Developer has a Checkstyle config without a complexity ceiling
        - When fitness runs checkstyle
            - Then it fails naming the config and the module to add
- 0079.4
    - Given Developer has a build and no Checkstyle config
        - When fitness runs checkstyle
            - Then it fails asking for a config with a complexity ceiling
- 0079.5
    - Given Developer has a build that does not declare Checkstyle
        - When fitness runs checkstyle
            - Then it fails with the plugin snippet for that build
- 0079.6
    - Given Developer has a repo with no Maven or Gradle build
        - When fitness runs checkstyle
            - Then it passes with zero files

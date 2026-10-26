---
relatedConfigurations: ['../../.fitnessrc.json']
---

<!-- cspell:ignore jacoco -->

# 0081 Checks Java Test

## Why

My Java tests run once and meet the same coverage goal as every other language. A failing test, a gap in coverage, or code with no tests shows up before I push.

## Measurement

Java repos whose tests pass and meet the coverage goal reported green
-
Java repos checked

## Requirements

- 0081.1
    - Given Developer has a build whose tests fail
        - When fitness runs java-test
            - Then it fails naming each failing test and its message
- 0081.3
    - Given Developer has a coverage report below the goal
        - When fitness runs java-test
            - Then it fails naming the report, the total and the goal
- 0081.4
    - Given Developer has a multi-module build with an aggregate coverage report
        - When fitness runs java-test
            - Then it judges the aggregate and each module report the aggregate does not cover
- 0081.5
    - Given Developer has product code whose build ran no test
        - When fitness runs java-test
            - Then it fails asking for tests, with each spotted framework's hint
- 0081.6
    - Given Developer has Testcontainers tests and no running Docker
        - When fitness runs java-test
            - Then it fails asking for Docker before the build runs
- 0081.7
    - Given Developer has a build without JaCoCo XML reports
        - When fitness runs java-test
            - Then it fails with the plugin snippet for that build
- 0081.8
    - Given Developer has a repo with no Maven or Gradle build
        - When fitness runs java-test
            - Then it passes with zero files
- 0081.9
    - Given Developer has passing unit and integration tests with full coverage
        - When fitness runs java-test
            - Then it passes, and Gradle restores unchanged tests from its build cache unless `gradle.properties` turns caching off

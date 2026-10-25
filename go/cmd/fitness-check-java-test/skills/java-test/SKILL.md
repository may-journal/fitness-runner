---
name: java-test
description: Fix a failing java-test check on Java tests and coverage. Use when a test fails, a build runs no tests, or JaCoCo coverage is below the goal.
---

<!-- cspell:ignore jacoco temurin -->

# Fix Java tests and coverage

The check runs the build's tests once per build root, then judges the results and the JaCoCo coverage. [The check README](../../README.md) lists the reports and frameworks it reads.

## Read the failure

- `AppTest.adds failed: expected 3 but was 2`: the form is `Class.method failed: message`. Fix the code or the test, then rerun.
- `target/site/jacoco/jacoco.xml: lines 8/10 covered; the goal is 100%`: write tests that run the uncovered lines or branches. Branches read the same way.
- `the Maven build at . has product code such as src/main/java/App.java and no tests; add tests under src/test/java`: write tests.
- `the Maven build at . has test files but ran no test`: the lines after it hint at the framework setup the tests lack.
- `no JaCoCo XML report after the Maven build at . ran its tests; add: ...`: add the JaCoCo snippet shown.
- `Java not installed: install a JDK, such as Temurin from https://adoptium.net, or set JAVA_HOME`: install a JDK.

For any other build error, read the build's own error lines.

## Check it first

Run `fitness java-test` from the repo root. Exit 0 means all tests pass at the coverage goal.

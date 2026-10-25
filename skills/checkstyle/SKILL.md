---
name: checkstyle
description: Fix a failing checkstyle check on Java code. Use when Checkstyle reports a violation or the repo lacks a complexity ceiling.
---

<!-- cspell:ignore temurin -->

# Fix Checkstyle findings

Java code passes the Checkstyle plugin the repo's own build declares, and the config holds a `CyclomaticComplexity` ceiling. [The check README](../../go/cmd/fitness-check-checkstyle/README.md) states the rules.

## Read the failure

- `src/main/java/App.java:12:5 - Method length is 80 (max allowed is 60) (MethodLength)`: the form is `file:line[:col] - message (Rule)`. Change the code at that line to satisfy the named rule, as by splitting the method.
- `no Checkstyle config sets a complexity ceiling; add config/checkstyle/checkstyle.xml holding ...`: add the config the message spells out.
- `<file> sets no complexity ceiling; add inside TreeWalker: ...`: add the `CyclomaticComplexity` module the message shows.
- `Checkstyle is not in the Maven build at <dir>; add it: ...`: add the plugin snippet shown. The Gradle wording names the Gradle build.
- `Java not installed: install a JDK, such as Temurin from https://adoptium.net, or set JAVA_HOME`: install a JDK.
- `Maven not installed: add the Maven wrapper (mvn wrapper:wrapper) or install Maven`: add the wrapper. Gradle has a matching message.

Fix the code. Do not loosen the rule or suppress the finding.

## Check it first

Run `fitness checkstyle` from the repo root. Exit 0 means no finding, warnings included.

# 0080 Checks Java Version

## Why

I learn at once when my repo pins no JDK or my JDK is older than its pin, before a build fails strangely.

## Measurement

Java repos whose running JDK satisfies every pin
-
Java repos checked

## Requirements

- 0080.1
    - Given Developer has a build that pins no JDK
        - When fitness runs java-version
            - Then it fails with the snippets that add a pin
- 0080.2
    - Given Developer has a version file at or below the running JDK
        - When fitness runs java-version
            - Then it passes
- 0080.3
    - Given Developer has a build release above the running JDK
        - When fitness runs java-version
            - Then it fails naming the file and the release it needs
- 0080.4
    - Given Developer has a Maven release set through a property, in the pom or a parent it inherits
        - When fitness runs java-version
            - Then it judges the property's release
- 0080.5
    - Given Developer has a pin that names no release
        - When fitness runs java-version
            - Then it fails naming the pin
- 0080.6
    - Given Developer has a build and no running JDK
        - When fitness runs java-version
            - Then it fails telling me to install a JDK
- 0080.7
    - Given Developer has a repo with no Maven or Gradle build
        - When fitness runs java-version
            - Then it passes with zero files
- 0080.8
    - Given Developer has a Gradle toolchain no installed JDK matches, and no resolver
        - When fitness runs java-version
            - Then it fails naming the JDK to install

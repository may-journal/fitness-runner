# 0140 Engine Code and Comment Views

## Why

Commenting out a line takes it out of my code for every check, while my comments are still judged as prose. The runner reads each file once and hands every check the same two views. No check decides on its own what a comment is.

## Measurement

Commented-out lines no code check counts
-
Commented-out lines in the repo

## Requirements

- 0140.1
    - Given Developer comments out an import from a dist folder in a TypeScript file
        - When build-output-untracked checks the repo with another check
            - Then it passes, since a commented-out import is no code
- 0140.2
    - Given Developer sets Vitest thresholds of 90 below a commented-out line setting them to 100
        - When vitest-coverage-full checks the repo
            - Then it fails saying every threshold must be 100
- 0140.3
    - Given Developer comments out the release a pom.xml pins and the line of a .java-version file
        - When java-version checks the repo
            - Then it says no JDK version is pinned
- 0140.4
    - Given Developer has a Go, Swift and TypeScript comment with a sentence past its limit
        - When prose-budget checks the repo with another check
            - Then it fails naming each comment's path and line
- 0140.5
    - Given Developer wraps code duplicated in two files in duplicate-code ignore markers
        - When duplicate-code checks the repo with another check
            - Then it passes, since the code view keeps the markers
- 0140.6
    - Given Developer has a workflow job with no summary step and a cspell directive after `jobs:`
        - When workflow-reporting checks the repo with another check
            - Then it names the job, since the directive is no YAML

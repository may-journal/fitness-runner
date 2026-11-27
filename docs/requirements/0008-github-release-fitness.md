# 0008 Release Fitness

## Why

I release fitness and fitness-ci from `version.txt` and checked archives, with notes that list each new entry and major tags that move.

## Measurement

releases whose tag, archives and notes match the repo

-

releases

## Requirements

1. Verifies the release version
   1. GIVEN Developer has a `version.txt`
      1. WHEN the release tool runs
         1. THEN it prints that version or names the mistake
         2. THEN it verifies that version or names the mistake
2. Summarizes each release step
   1. GIVEN Developer has a release step
      1. WHEN it passes or fails
         1. THEN a GitHub job summary shows its outcome
         2. THEN the summary shows only inside Actions
3. Hashes each platform archive
   1. GIVEN Developer has built release archives
      1. WHEN `bundle-hashes` or `assemble` or `record` runs
         1. THEN each platform's hash prints
         2. THEN each platform's size prints
         3. THEN a changed file fails
4. Writes notes from entry files
   1. GIVEN Developer has changelog entry files
      1. WHEN a release writes its notes
         1. THEN they list each new entry newest first
         2. THEN they hold a working compare link
5. Links the tracking issue once
   1. GIVEN Developer has a release pull request
      1. WHEN Release Please runs
         1. THEN it links the tracking issue once
         2. THEN it skips the steps a release lacks
6. Moves each major tag to its release
   1. GIVEN Developer has a created release
      1. WHEN Release Please moves the major tags
         1. THEN each tag points at its own release or nothing moves
7. Runs each workflow subcommand
   1. GIVEN Developer has the workflows of this repo
   2. GIVEN Developer has the actions of this repo
      1. WHEN each step runs its fitness-ci subcommand the way its workflow does
         1. THEN the subcommand exists
         2. THEN the subcommand does the step's job
8. Declares each step output
   1. GIVEN Developer has a subcommand that a reusable workflow runs through the fitness-ci action
      1. WHEN the subcommand writes a step output
         1. THEN the action declares that output so the caller reads it

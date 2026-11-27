# 0001 Install and Update Fitness

## Why

I get the exact, verified fitness and its skills with one installer, and my old files move to the current shape.

## Measurement

installs that run the verified release they asked for

-

installs

## Requirements

1. Runs the release it was asked for
   1. GIVEN Developer has any folder
      1. WHEN I run `fitness-install` with its options
         1. THEN it runs the verified release or says why it cannot
         2. THEN it exits 1 if it cannot run the release
2. Publishes the install to a GitHub job
   1. GIVEN Developer has a GitHub job
      1. WHEN it runs `fitness-install --github-action`
         1. THEN it writes the folder to the job outputs
         2. THEN it summarizes the result
         3. THEN it refuses a release outside the action's major ref
3. Looks up the latest release politely
   1. GIVEN Developer asks for the latest release
      1. WHEN `fitness-install` looks it up
         1. THEN it sends my token
         2. THEN it asks once per 10 minutes
         3. THEN it survives a refusal
4. Links skills into an agent folder
   1. GIVEN Developer has an agent's skills folder
      1. WHEN I run `fitness-install --link-skills` with it
         1. THEN every skill in the release links there readable
         2. THEN my own skills stay
5. Ships a complete onboarding skill
   1. GIVEN Developer has the onboarding skill linked
      1. WHEN I follow it
         1. THEN it covers adopting fitness
         2. THEN every doc it links exists
6. Adds hooks to Claude settings once
   1. GIVEN Developer has a Claude Code settings file of my own
      1. WHEN I run `fitness-install --claude-settings` with it twice
         1. THEN it holds each fitness hook once
         2. THEN my settings stay
7. Strips old front matter
   1. GIVEN Developer has docs with old fitness front matter
      1. WHEN `fitness strip-front-matter` runs
         1. THEN only the fitness keys go
         2. THEN every body stays
8. Migrates ADR files to issues
   1. GIVEN Developer has ADR files
      1. WHEN `fitness adr-migrate` runs
         1. THEN each becomes one `ADR` issue
         2. THEN none is lost or opened twice

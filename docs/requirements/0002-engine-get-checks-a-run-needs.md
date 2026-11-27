# 0002 Get the Checks a Run Needs

## Why

Each run gets the checks my repo pins, verified, downloading only what it needs and working offline.

## Measurement

runs whose every check came from a verified pinned release

-

runs

## Requirements

1. Lists each pin
   1. GIVEN Developer has a repo's `checkVersions`
      1. WHEN I run `fitness check-versions`
         1. THEN it prints each check with its pin or names a bad pin
2. Installs pinned checks
   1. GIVEN Developer has a repo that pins checks
      1. WHEN I run `fitness-install --with-checks`
         1. THEN each pin installs verified into one folder or a warning names it
3. Finds installed checks
   1. GIVEN Developer has check programs installed
      1. WHEN fitness runs them
         1. THEN it finds each
         2. THEN it agrees its protocol
         3. THEN it names how to install any missing one
4. Downloads only the checks a run reaches
   1. GIVEN Developer has a repo pinning checks
   2. GIVEN none are cached
      1. WHEN I run fitness on some files
         1. THEN it downloads only the checks those files reach
         2. THEN it verifies only the checks those files reach
5. Downloads every check when asked
   1. GIVEN Developer has a repo pinning checks
      1. WHEN I run `fitness --all` or `fitness prefetch` or a command that needs every check
         1. THEN every pinned check downloads up front
6. Runs cached checks offline
   1. GIVEN Developer has no network
      1. WHEN I run fitness
         1. THEN cached checks run
         2. THEN each uncached one says it is offline
7. Prunes unused releases
   1. GIVEN Developer has a cache of unused releases
      1. WHEN I run fitness with GitHub answering
         1. THEN it deletes only what nothing uses or links or names

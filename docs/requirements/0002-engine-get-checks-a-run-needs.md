# 0002 Get the Checks a Run Needs

## Why

Each run gets the checks my repo pins, verified, downloading only what it needs and working offline.

## Measurement

runs whose every check came from a verified pinned release

-

runs

## Requirements

- 0002.1
  - Given Developer has a repo's `checkVersions`
    - When I run `fitness check-versions`
      - Then it prints each check with its pin, or names a bad pin
- 0002.2
  - Given Developer has a repo that pins checks
    - When I run `fitness-install --with-checks`
      - Then each pin installs verified into one folder, or a warning names it
- 0002.3
  - Given Developer has check programs installed
    - When fitness runs them
      - Then it finds each, agrees its protocol, and names how to install any missing one
- 0002.4
  - Given Developer has a repo pinning checks, none cached
    - When I run fitness on some files
      - Then it downloads and verifies only the checks those files reach
- 0002.5
  - Given Developer has a repo pinning checks
    - When I run `fitness --all`, `fitness prefetch` or a command that needs every check
      - Then every pinned check downloads up front
- 0002.6
  - Given Developer has no network
    - When I run fitness
      - Then cached checks run, and each uncached one says it is offline
- 0002.7
  - Given Developer has a cache of unused releases
    - When I run fitness with GitHub answering
      - Then it deletes only what nothing uses, links or names

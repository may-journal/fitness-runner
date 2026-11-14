# 0163 Engine check versions

## Why

Each check releases on its own line. So a repo pins the checks it uses, and the installer reads one list of checks and pins.

## Measurement

Check pins the engine prints or refuses as documented
-
Check pins

## Requirements

- 0163.1
    - Given Developer has a repo whose `.fitnessrc.json` sets no `checkVersions`, or an empty one
        - When I run `fitness check-versions`
            - Then it prints every check fitness-checks releases at `v1`, one per line, and no check the engine judges itself
- 0163.2
    - Given Developer has a repo whose `checkVersions` pins one check to a release and one to a major
        - When I run `fitness check-versions`
            - Then it prints exactly those two checks with their pins
- 0163.3
    - Given Developer has a repo whose `checkVersions` holds a malformed pin or name, or pins a check the engine runs
        - When I run `fitness check-versions`
            - Then it fails, naming the check and how to fix it

# 0060 Checks Vitest Coverage Full

## Why

I trust every line of my JavaScript code ran under a test, because coverage below 100% turns the check red.

## Measurement

JavaScript projects whose Vitest coverage run passes at 100% thresholds
-
JavaScript projects checked

## Requirements

- 0060.1
    - Given Developer has a repo without JavaScript or TypeScript source, even with a `package.json` for formatting tools
        - When vitest-coverage-full runs
            - Then it passes having checked 0 files
- 0060.2
    - Given Developer has a Vitest config with a coverage threshold below 100
        - When vitest-coverage-full runs
            - Then it fails asking for thresholds of 100
- 0060.3
    - Given Developer has an installed shared config with a threshold below 100
        - When vitest-coverage-full runs
            - Then it fails asking the shared package for thresholds of 100
- 0060.4
    - Given Developer has fully tested code with Vitest installed and thresholds at 100
        - When vitest-coverage-full runs
            - Then it passes
- 0060.5
    - Given Developer has code that no test runs
        - When vitest-coverage-full runs
            - Then it fails showing the coverage Vitest reported
- 0060.6
    - Given Developer has a JavaScript project without Vitest installed
        - When vitest-coverage-full runs
            - Then it shows info with a hint to install Vitest, never failing
- 0060.7
    - Given Developer has a non-executable `node_modules/.bin/vitest` and Vitest on `PATH`
        - When vitest-coverage-full runs
            - Then it runs the Vitest on `PATH` and passes
- 0060.8
    - Given Developer has a Vitest that exits non-zero with no threshold miss or failed test
        - When vitest-coverage-full runs
            - Then it shows info quoting Vitest's last lines, never failing

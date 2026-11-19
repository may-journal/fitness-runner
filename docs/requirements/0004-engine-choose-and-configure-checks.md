# 0004 Choose and Configure Checks

## Why

My repo chooses which checks run, adds its own, and sees why each runs, while the org bar stays whole.

## Measurement

checks whose run matches the repo's settings

-

checks

## Requirements

- 0004.1
  - Given Developer has a check of the repo's own
    - When the suite runs
      - Then it runs beside the bundled checks and reports its verdict
- 0004.2
  - Given Developer has external policy
    - When a setting replaces or turns off a bundled check
      - Then the run follows it and names the replaced check
- 0004.3
  - Given Developer has org policy
    - When a setting replaces or turns off a check
      - Then the run is red, naming the org rule
- 0004.4
  - Given Developer has a check setting or folder fitness cannot use
    - When the suite starts
      - Then it is red, naming the setting and the fix
- 0004.5
  - Given AI agent has a repo
    - When `fitness config` runs
      - Then it shows each setting, its source, and why each check runs or skips
- 0004.6
  - Given Developer has a workflow job with a long timeout
    - When `workflow-timeouts` runs
      - Then it passes only a timeout that cites its observed maximum

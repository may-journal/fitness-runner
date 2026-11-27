# 0004 Choose and Configure Checks

## Why

My repo chooses which checks run, adds its own, and sees why each runs, while the org bar stays whole.

## Measurement

checks whose run matches the repo's settings

-

checks

## Requirements

1. Runs the repo's own checks
   1. GIVEN Developer has a check of the repo's own
      1. WHEN the suite runs
         1. THEN it runs beside the bundled checks
         2. THEN it reports its verdict
2. Follows external policy
   1. GIVEN Developer has external policy
      1. WHEN a setting replaces or turns off a bundled check
         1. THEN the run follows it
         2. THEN the run names the replaced check
3. Holds the org bar
   1. GIVEN Developer has org policy
      1. WHEN a setting replaces or turns off a check
         1. THEN the run is red naming the org rule
4. Rejects settings it cannot use
   1. GIVEN Developer has a check setting or folder fitness cannot use
      1. WHEN the suite starts
         1. THEN it is red naming the setting
         2. THEN it is red naming the fix
5. Shows the resolved config
   1. GIVEN AI agent has a repo
      1. WHEN `fitness config` or its schema runs even before any check is installed
         1. THEN it shows each setting
         2. THEN it shows every check's well-formed declared keys among them
         3. THEN it shows each setting's source
         4. THEN it shows why each check runs or skips
6. Cites each workflow timeout
   1. GIVEN Developer has a workflow job with a long timeout
      1. WHEN `workflow-timeouts` runs
         1. THEN it passes only a timeout that cites its observed maximum
7. Fails a required check that cannot judge
   1. GIVEN Developer names a check required in `requiredChecks` or with `--require`
      1. WHEN that check cannot judge
         1. THEN the run is red naming the check
         2. THEN other checks still show info
8. Removed in 2026.11.27.2030: moved to go/pkg/lang/node as 0001.2

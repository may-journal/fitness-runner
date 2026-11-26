# 0003 Run Checks and Read Results

## Why

I run every check, one, or a folder. I see what broke, where, and how to fix it, in the terminal, as JSON or in Actions.

## Measurement

runs whose every check ends green, red or info

-

runs

## Requirements

1. Ends each check with a verdict
   1. GIVEN Developer has a repo
      1. WHEN fitness runs
         1. THEN each check ends green or red or info
         2. THEN the exit code says whether the bar holds
2. Names what to fix
   1. GIVEN Developer asks fitness for something it cannot do
      1. WHEN fitness runs
         1. THEN it names what to fix
         2. THEN it exits nonzero
3. Runs only the checks an input reaches
   1. GIVEN Developer has checks that judge only some files
      1. WHEN I run fitness on a folder or a text or the repo
         1. THEN only the checks the input reaches run
         2. THEN each skip says why
4. Overlaps slow checks
   1. GIVEN Developer has slow checks
      1. WHEN fitness runs
         1. THEN they overlap
         2. THEN a check past its stage's total stops named with its time
5. Shows findings in Actions
   1. GIVEN Developer has a run in GitHub Actions
      1. WHEN fitness reports it
         1. THEN annotations and the job summary show each finding where it is
6. Reports results as JSON
   1. GIVEN AI agent wants results as data
      1. WHEN fitness runs with `--format=json` or `--json`
         1. THEN each finding comes back as JSON
         2. THEN each check's cost comes back as JSON
7. Explains itself from the code
   1. GIVEN AI agent has fitness installed
      1. WHEN it runs `fitness list` or `explain` or `docs` or `llms` or `--help`
         1. THEN the binary explains each command from its comment
         2. THEN the binary explains each page from go/docs Markdown
         3. THEN a test fails drift from the code
8. Judges comments as prose
   1. GIVEN Developer has code with comments
      1. WHEN `fitness comments` or a check reads it
         1. THEN comments are judged as prose
         2. THEN commented-out code is no code

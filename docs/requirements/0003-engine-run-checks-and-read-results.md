# 0003 Run Checks and Read Results

## Why

I run every check, one, or a folder. I see what broke, where, and how to fix it, in the terminal, as JSON or in Actions.

## Measurement

runs whose every check ends green, red or info

-

runs

## Requirements

- 0003.1
  - Given Developer has a repo
    - When fitness runs
      - Then each check ends green, red or info, and the exit code says whether the bar holds
- 0003.2
  - Given Developer asks fitness for something it cannot do
    - When fitness runs
      - Then it names what to fix and exits nonzero
- 0003.3
  - Given Developer has checks that judge only some files
    - When I run fitness on a folder, a text or the repo
      - Then only the checks the input reaches run, and each skip says why
- 0003.4
  - Given Developer has slow checks
    - When fitness runs
      - Then they overlap, and a check past its stage's total stops, named with its time
- 0003.5
  - Given Developer has a run in GitHub Actions
    - When fitness reports it
      - Then annotations and the job summary show each finding where it is
- 0003.6
  - Given AI agent wants results as data
    - When fitness runs with `--format=json` or `--json`
      - Then each finding and each check's cost comes back as JSON
- 0003.7
  - Given AI agent has fitness installed
    - When it runs `fitness list`, `explain`, `docs`, `llms` or `--help`
      - Then the binary explains each command from its comment and each page from go/docs Markdown, and a test fails drift from the code
- 0003.8
  - Given Developer has code with comments
    - When `fitness comments` or a check reads it
      - Then comments are judged as prose, and commented-out code is no code

# 0171 GitHub Workflow Scripts

## Why

Shell inside a workflow cannot be linted or tested. So each step that does real work calls a checked-in script, and a typo in one is caught before a run.

## Measurement

Workflow and action steps that call a script that exists and is executable
-
Workflow and action steps that call a script

## Requirements

- 0171.1
    - Given Developer has the workflows and actions of this repo
        - When each reference to a script is read
            - Then every script it names exists and is executable
- 0171.2
    - Given Developer has a script that a reusable workflow runs through the script action
        - When the script writes a step output
            - Then the script action declares that output, so the caller reads it

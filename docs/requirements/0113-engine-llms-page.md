# 0113 LLMs Page

## Why

Any AI that meets fitness reads one page and knows what it is, what to run and how to read the result. The page and `fitness --help` come from the code's comments, so neither misses a command.

## Measurement

subcommands the llms page names
-
subcommands fitness runs

## Requirements

- 0113.1
    - Given AI agent has a repo with fitness installed
        - When it runs `fitness llms`
            - Then the page explains fitness, its exit codes, every subcommand, every built-in check and every docs topic
- 0113.2
    - Given AI agent asks for help
        - When it runs `fitness --help`
            - Then every subcommand the llms page names is listed, and the help points at `fitness llms`

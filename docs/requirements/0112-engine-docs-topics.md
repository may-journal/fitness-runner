# 0112 Docs Topics

## Why

I read how checks talk to the runner, the config keys and the GitHub commands from the installed binary. Each topic is the code's own doc comments.

## Measurement

topics `fitness docs` prints
-
topics `fitness docs` lists

## Requirements

- 0112.1
    - Given AI agent has a repo with fitness installed
        - When it runs `fitness docs` and then `fitness docs` with each listed topic
            - Then each topic prints its title and the doc comments of its package
- 0112.2
    - Given AI agent asks for a topic that does not exist
        - When it runs `fitness docs nope`
            - Then fitness exits 2 and lists the topics
- 0112.3
    - Given AI agent passes `docs` a flag, as `list` and `explain` take `--json`
        - When it runs `fitness docs --json`
            - Then fitness exits 2 and shows how to name a topic

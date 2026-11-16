# 0172 Engine checks that apply

## Why

A check judges only some files, so a run that reaches none of them gains nothing from it. Each check declares which files reach it and what else it requires. The engine leaves out the checks a run cannot use, saying why.

## Measurement

Installed checks the engine leaves out or stops on, as their declared fields say
-
Installed checks that declare fields

## Requirements

- 0172.1
    - Given Developer has an installed check whose `files` pattern finds only Go files, and one that finds Markdown
        - When I run both scoped to a folder of Markdown, then across the whole repo
            - Then the scoped run skips the Go check, naming it under no matching files, and the full run runs both
- 0172.2
    - Given Developer has an installed check whose `needs` pattern names a `package.json` the repo does not track
        - When I run it
            - Then the run skips it, naming it under the needs pattern
- 0172.3
    - Given Developer has an installed check whose `files` pattern does not compile, and one that requires what this fitness does not know
        - When I run each beside another check
            - Then the first stops the run, naming the field and install command, and the second is skipped, naming the upgrade
- 0172.4
    - Given Developer has two installed checks that judge a text, one of which requires the repo
        - When I pipe a text to `fitness --blob` with both
            - Then the run judges the text with the first and skips the second, naming what it requires
- 0172.5
    - Given Developer has an installed check that passes and one the run skips
        - When I run both, with and without `--format=json`
            - Then the skipped check is a skipped row, counted among the passes, and its JSON entry says why

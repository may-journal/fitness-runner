# 0166 Engine clean install

## Why

The built-in checks go away once each check ships on its own. So a clean install of the engine and its pinned checks must judge a repo as the built-ins do.

## Measurement

Pinned checks whose clean-install result matches the built-in result
-
Pinned checks

## Requirements

- 0166.1
    - Given Developer has a repo that pins three checks, with files that break two of them
        - When I run them through `fitness-install --with-checks` with `--no-builtins`, and again with the built-ins
            - Then both runs print the same results table and summary line by line, times and their padding aside

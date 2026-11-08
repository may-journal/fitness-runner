# 0156 Renovate's hidden debug block passes the spell check

## Why

A dependency PR from Renovate ends its body with a hidden debug block of encoded text. The spell check skips it, so the PR is not failed every hour for words no one wrote.

## Measurement

Renovate PR bodies whose only unknown words are in the debug block, which cspell passes

-

Renovate PR bodies whose only unknown words are in the debug block

## Requirements

- 0156.1
  - Given Developer has a fitness repo and a Renovate PR body that ends with its hidden debug block
    - When I run `fitness --blob -` with cspell on that body
      - Then cspell passes, and a misspelled word outside the block still fails

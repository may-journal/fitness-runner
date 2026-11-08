# 0149 Checks Prose Speed

## Why

The prose checks read my docs and comments once each and start no tool per file. A repo with hundreds of docs never spends the fast total on them.

## Measurement

full prose check runs on hundreds of files that pass within five seconds without a git call per file

-

full prose check runs on hundreds of files

## Requirements

- 0149.1
  - Given Developer has a repo with 300 docs and commented Go files
    - When text-readability, prose-budget, cspell, mermaid-callouts and repeated-string-literals check every file
      - Then each passes within five seconds, and the whole run starts git at most 20 times

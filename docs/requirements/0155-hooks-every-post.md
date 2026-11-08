# 0155 The hook stops every post that breaks the rules

## Why

Whatever folder Claude works in, and however it posts, a comment or body that breaks the rules never reaches GitHub. Each one meets the checks GitHub gives it, under the rules of the repo it goes to.

## Measurement

posts from Claude Code that the hook judges by the target repo's rules, or denies unread

-

posts from Claude Code

## Requirements

- 0155.1
  - Given Developer has a fitness repo whose `cspell.json` allows a word, and Claude drafted a comment with it
    - When Claude Code runs `fitness claude-hook` from another folder before `cd` into the repo and a `gh` call
      - Then it exits 0 with no output
- 0155.2
  - Given Developer has a fitness repo whose `cspell.json` allows a word, and Claude drafted a comment with it
    - When Claude Code runs `fitness claude-hook` before a `gh` call or script naming another repo, or outside any repo
      - Then it denies the call with the org's cspell findings
- 0155.3
  - Given Developer has a fitness repo
    - When Claude Code runs `fitness claude-hook` before a post whose body is a variable, a pipe or a file the line writes
      - Then it denies the call, saying how to pass a body it can read
- 0155.4
  - Given Developer has a fitness repo and Claude drafted a red comment
    - When Claude Code runs `fitness claude-hook` before a later `gh` call, a loop, `bash -c` or a wrapper such as `sudo` posts it
      - Then it denies the call with the findings
- 0155.5
  - Given Developer has a fitness repo and Claude drafted a red comment
    - When Claude Code runs `fitness claude-hook` before a GitHub MCP tool or a review reply posts it
      - Then it denies the call with the findings
- 0155.6
  - Given Developer has a fitness repo and Claude drafted a PR body or Plan with no Background section
    - When Claude Code runs `fitness claude-hook` before `gh` or a GitHub MCP tool creates it
      - Then it denies the call with the structure check's findings

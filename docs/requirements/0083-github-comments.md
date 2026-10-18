---
relatedConfigurations: ['../../.fitnessrc.json']
---

# 0083 Comments

## Why

I see a sloppy comment flagged before it posts, and on GitHub I see one reply saying what to fix.

## Measurement

failing comments with exactly one reply that names their violations
-
failing comments

Source: `go/internal/aftereffect/comment.go:56`

## Requirements

- 0083.1
    - Given Developer has a drafted comment with a change-log note and a callout table missing a diagram number
        - When I run fitness comment-check with the draft after `--blob` or on stdin
            - Then it names both violations with their lines and exits 1
- 0083.2
    - Given Developer has a drafted comment whose numbered diagram matches its callout table
        - When I run fitness comment-check with the draft after `--blob`
            - Then every check passes and it exits 0
- 0083.3
    - Given Developer has a failing comment on an issue with no reply yet
        - When comment-check judges the comment event
            - Then it posts one reply mentioning the author with the violations and the `--blob` re-check, reacts thumbs-down, and exits 1
- 0083.4
    - Given Developer has a failing comment whose reply already states the same violations
        - When comment-check judges it again
            - Then it writes no comment
- 0083.5
    - Given Developer has a failing comment that is then fixed
        - When comment-check judges the edit
            - Then it turns its reply into a passing note and removes its reaction
- 0083.6
    - Given Developer has comments from a known bot, a fitness verdict, and an author a caller adds to its ignore list
        - When comment-check judges each comment event
            - Then it skips them all and exits 0
- 0083.7
    - Given Developer has a failing review comment in a pull request thread
        - When comment-check judges the review comment event
            - Then it replies in that thread
- 0083.8
    - Given Developer has a manual workflow run
        - When comment-check runs
            - Then it judges the issue and review comments from the last seven days

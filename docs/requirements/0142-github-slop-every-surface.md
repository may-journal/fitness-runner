# 0142 GitHub Slop Every Surface

## Why

Whatever I write on GitHub meets the slop rules my code comments meet. That covers an issue or PR body, a comment, a review reply and a commit message.

## Measurement

GitHub texts and commit messages free of slop
-
GitHub texts and commit messages judged

## Requirements

- 0142.1
    - Given Developer has a PR body draft that says "Note that", hedges with "basically" and reframes as "It's not X. It's Y."
        - When I run fitness with the draft after `--blob`
            - Then prose-budget names the tutorial opener and the hedge, and no-contrastive-reframing names the reframing
- 0142.2
    - Given Developer has a review reply draft that says "Let's" and "It's not slow, but careful"
        - When I run fitness comment-check with the reply after `--blob`
            - Then it fails naming `tutorial-tone` and the reframing
- 0142.3
    - Given Developer commits with a message whose body says "Previously it used to" and "It's not a fix. It's a feature."
        - When I commit
            - Then the commit-msg hook fails naming `change-narration` and the reframing

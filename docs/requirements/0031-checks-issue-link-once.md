---
relatedConfigurations: ['../../.fitnessrc.json']
---

# 0031 Checks Issue Link Once

## Why

I read each issue's timeline without noise, because my branch links an issue from one commit, however I write the link.

## Measurement

repeat issue links reported red
-
repeat issue links

Source: `go/cmd/fitness-check-issue-link-once/main.go:127`

## Requirements

- 0031.1
    - Given a branch commit that links `#12` of origin's repo
        - When issue-link-once checks a message linking that issue by URL
            - Then it fails naming the earlier commit
- 0031.2
    - Given a branch commit that links `#12` of origin's repo
        - When issue-link-once checks a message linking `#12` of another repo
            - Then it passes
- 0031.3
    - Given a message whose repeat link sits in a comment or below the scissors line
        - When issue-link-once checks the message
            - Then it passes, as git drops those lines
- 0031.4
    - Given the branch's open pull request body links an issue
        - When issue-link-once checks a message linking that issue
            - Then it fails naming the open pull request
- 0031.5
    - Given a pull request whose one early commit links its issue, with `main` merged in
        - When issue-link-once runs on that pull request in CI
            - Then it passes
- 0031.6
    - Given a pull request where a second commit repeats a link
        - When issue-link-once runs on that pull request in CI
            - Then it fails naming that commit and the force-push fix
- 0031.7
    - Given a commit written after the pull request opened, linking the issue its body links
        - When issue-link-once runs on that pull request in CI
            - Then it fails naming the open pull request
- 0031.8
    - Given no message and no pull request
        - When I run issue-link-once
            - Then it passes having checked nothing

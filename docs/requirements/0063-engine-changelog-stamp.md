---
relatedConfigurations: ['../../.fitnessrc.json']
---

# 0063 Changelog Stamp

## Why

I write a changelog entry file under any placeholder heading. My first commit on a branch stamps it with the real time.

## Measurement

commits whose entry file heading shows the time of the first commit adding it
-
commits that stage an entry file

Source: `go/cmd/fitness-stamp-changelog/main.go:40`

## Requirements

- 0063.5
    - Given Developer has the hooks and stages a new entry file under an old heading
        - When I commit it
            - Then the committed entry file shows the commit time
- 0063.6
    - Given Developer has the hooks and a branch whose earlier commit added its entry file
        - When I commit more bullets in that file
            - Then the committed heading keeps the first commit's time

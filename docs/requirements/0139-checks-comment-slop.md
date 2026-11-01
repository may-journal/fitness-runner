# 0139 Checks Comment Slop

## Why

My code comments hold no dead code, echoes of the next line, decoration, history or filler, because prose-budget fails each one. Markdown files are judged as before.

## Measurement

Comment blocks free of slop
-
Comment blocks judged

## Requirements

- 0139.1
    - Given Developer has commented-out code in Go, Swift and TypeScript
        - When prose-budget checks the repo
            - Then it fails each block as `commented-out-code` and says to delete it
- 0139.2
    - Given Developer has the comment "increment i" above code that increments i
        - When prose-budget checks the repo
            - Then it fails each comment as `restates-code`
- 0139.3
    - Given Developer has an emoji comment and a banner line of equals signs
        - When prose-budget checks the repo
            - Then it fails each as `decoration` and names what to delete
- 0139.4
    - Given Developer has a comment saying how code "previously" worked
        - When prose-budget checks the repo
            - Then it fails as `change-narration` and says to describe what holds today
- 0139.5
    - Given Developer has a comment that "basically" states a fact
        - When prose-budget checks the repo
            - Then it fails as `hedging` and says to drop the word
- 0139.6
    - Given Developer has comments with "just" as only, an arrow, "no longer" or "previously stored", and sloppy Markdown
        - When prose-budget checks the repo
            - Then it passes
- 0139.7
    - Given Developer stages a hedging comment in a file whose other comment hedges too
        - When prose-budget runs on the staged hedge
            - Then it judges only the changed comment, and a full scan lists both

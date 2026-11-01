# 0141 Checks Comment Tutorial Tone

## Why

My code comments state rules plainly instead of walking a reader through code, because prose-budget fails a tutorial opener. Markdown files are judged as before.

## Measurement

Comment blocks free of tutorial openers
-
Comment blocks judged

## Requirements

- 0141.1
    - Given Developer has comments that open a sentence with "Note that", "Let's", "As you can see" or "First we"
        - When prose-budget checks the repo
            - Then it fails each as `tutorial-tone` and says to state the rule plainly
- 0141.2
    - Given Developer has comments holding "a note that", "lets", or a quoted "Let's go", and a Markdown file saying "Note that"
        - When prose-budget checks the repo
            - Then it passes

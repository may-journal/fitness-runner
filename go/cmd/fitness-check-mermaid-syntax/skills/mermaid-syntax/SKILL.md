---
name: mermaid-syntax
description: Fix a failing mermaid-syntax check on a mermaid block. Use when a diagram would not render or a Plan, PR, or comment body holds a broken diagram.
---

# Make the diagram parse

Every mermaid block must parse, so a broken diagram never reaches a reader. [The check README](../../README.md) lists what each diagram type is checked for.

## Read the failure

Each error reads `<file>: line <n>: <problem> (diagram at line <m>)`. The line is in the document.

- `label "a (b)" has brackets or quotes; wrap it in double quotes`: write the label as `"a (b)"`.
- `"[" is never closed` or `"]" closes nothing`: balance the brackets.
- `"x" is not a flowchart header; use flowchart TD, LR, BT, or RL`: fix the first line.
- `expected an edge such as --> after a node, got "x"`: add the arrow, or fix the stray text.
- `message "x" needs a target and a colon, as in A->>B: text`: fix the sequence message.
- `unknown diagram type "x"`: use a supported type.

## Check it first

Run `fitness mermaid-syntax` from the repo root for all docs. To test a body, add `--body-file path`. Exit 0 means every diagram parses.

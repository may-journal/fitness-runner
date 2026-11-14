# 0159 Checks Questions In Chat

## Why

I see every question an agent has for me in chat, since no doc, page or body holds a list of them.

## Measurement

docs, pages and bodies with no list of open questions

-

docs, pages and bodies Fitness judges

## Requirements

- 0159.1
  - Given Developer has a doc with a heading, bold line or label line, even in a list item, naming open questions
    - When open-questions checks the repo
      - Then it fails at each line, saying to ask in chat
- 0159.2
  - Given Developer has a doc that mentions open questions only in a sentence, a bold sentence or code
    - When open-questions checks the repo
      - Then it passes
- 0159.3
  - Given Developer has an HTML page whose `<h2>` heading names open questions
    - When open-questions checks the repo
      - Then it fails naming the page and line
- 0159.4
  - Given Developer has a PR body and a Plan body, each with an `Open questions:` line
    - When pr-check and issue-check judge them
      - Then both fail with the open-questions finding
- 0159.5
  - Given AI agent has an Artifact page whose heading names open questions
    - When Claude Code runs `fitness claude-hook` before the page publishes
      - Then it denies the publish naming the line

# 0123 Checks No Fitness Front Matter

## Why

No check reads the old fitness keys, so a block left behind only teaches agents to copy it. I want any leftover flagged, with the fixer named.

## Measurement

Markdown files holding fitness front matter reported red
-
Markdown files holding fitness front matter

## Requirements

- 0123.1
    - Given Developer has docs whose front matter holds a fitness key, plain or in an HTML comment
        - When fitness runs no-fitness-front-matter
            - Then it fails once per file, naming each file and `fitness strip-front-matter`
- 0123.2
    - Given Developer has docs whose fitness front matter has CRLF endings or a byte order mark, beside another tool's keys
        - When fitness runs no-fitness-front-matter
            - Then it fails naming each file
- 0123.3
    - Given Developer has docs with no front matter, another tool's front matter, a leading rule, or fixture repos holding fitness keys
        - When fitness runs no-fitness-front-matter
            - Then it passes

### 2026.11.02.1655

- Feat: the mermaid checks judge a fenced diagram inside a blockquote, as GitHub renders it, in docs and in GitHub bodies.
- Feat: the mermaid checks read `.mmd` and `.mermaid` files as one diagram each; Markdown checks read every Markdown ending from one list.
- Fix: `mermaid-callouts` fails an unnumbered diagram of a type with labels, unless it is a legend; `mermaid-diagram-prose` judges diagrams without a table.
- Feat: a wardley-beta `annotation N,[…]` line counts as callout N; requirements 0084.9, 0036.8, 0039.13, 0039.14 and 0040.9 prove each change by real runs.
- Fix: a quoted fence ends with its quote, a quoted example in code stays code, and `--fix` edits a quoted row in place.

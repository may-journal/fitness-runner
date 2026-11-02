### 2026.11.02.1655

- Feat: the mermaid checks judge a fenced diagram inside a blockquote, as GitHub renders it, in docs and in GitHub bodies.
- Feat: the mermaid checks read `.mmd` and `.mermaid` files as one diagram each, and name their lines without an opening fence.
- Feat: every Markdown check reads each ending GitHub renders as Markdown, plus MDX, from one shared list.
- Test: requirements 0084.9 and 0036.8 prove a quoted fence, a `.markdown` doc, a `.mmd` file and an `.mdx` doc by real runs.

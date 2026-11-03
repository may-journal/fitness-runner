### 2026.11.02.1830

- Fix: `duplicate-code` never counts a legend diagram, a style statement in a mermaid block, or a table separator row.
- Fix: a commit scoped to Markdown passes the architecture docs, as a full scan does.
- Fix: a long table separator row no longer reads as a block of copied code.
- Test: requirement 0032.16 proves a commit scoped to Markdown passes the legend and style lines a doc must repeat, and 0032.17 the separator rows.

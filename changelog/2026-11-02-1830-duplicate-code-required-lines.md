### 2026.11.02.1830

- Fix: `duplicate-code` never counts a diagram titled Legend or under a Legend heading, a mermaid style statement, or a table's separator row.
- Fix: a commit scoped to Markdown passes the architecture docs, as a full scan does.
- Fix: a `class` line in a class diagram, a `---` break and a dash line in fenced code still count.
- Test: requirement 0032.16 proves a scoped commit skips legends, style lines and templates but names a copied diagram, and 0032.17 the separator rows.
- Test: 0032.17 holds the minimums that the released 2026.10.26.2140 entry credits to 0032.12, an ID now retired.

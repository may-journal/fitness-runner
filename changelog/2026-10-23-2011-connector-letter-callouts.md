### 2026.10.23.2011

- Feat: `mermaid-callouts` reads node callouts as numbers and connector callouts as letters: `A` to `Z`, then `AA` to `ZZ`, then `AAA`.
- Feat: `mermaid-callouts` fails a numbered connector, and a letter that breaks the doubling, such as `AB`.
- Feat: `mermaid-callout-why` asks a Why for letter rows too, and the architecture checks bind letter rows to their arrows.
- Docs: the architecture views and the shipped templates label arrows with letters.
- Test: requirement 0039 covers letter callouts, numbered connectors, and broken doubling.

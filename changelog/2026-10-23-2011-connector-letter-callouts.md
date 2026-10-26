### 2026.10.23.2011

- Feat: `mermaid-callouts` reads node callouts as numbers and connector callouts as letters: `A` to `Z`, then `AA` to `ZZ`, then `AAA`.
- Feat: `mermaid-callouts` fails a numbered connector, and a listed letter that breaks the doubling, such as `AB`.
- Feat: `mermaid-callout-why`, `mermaid-diagram-prose`, and the architecture checks read letter callouts too.
- Docs: the architecture views and the shipped templates label arrows with letters.
- Test: requirements 0039 and 0040 cover letters, broken doubling, prose labels such as `YES` and `HTTPS`, and letter prose.

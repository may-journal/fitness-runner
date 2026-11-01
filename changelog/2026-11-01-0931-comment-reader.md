### 2026.11.01.0931

- Feat: one shared reader returns each comment block of nine languages, with its path, line, language and kind.
- Feat: `fitness comments [path...]` prints the blocks of tracked files, skipping tool directives, a license header and generated files.
- Refactor: `codemodel.Scanner` and the Swift blanker find comments through the shared reader, and strip what they stripped before.
- Test: requirement 0136 proves by real runs that a `//` inside a string, template or raw string is no comment.
- Test: it also proves block, doc, shell and YAML comments come back at their line, and folder paths work.

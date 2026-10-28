### 2026.10.26.1900
- Feat: a requirement doc's Measurement is a numerator, `-` and a denominator, with no `Source:` line.
- Feat: a leftover `Source:` line is no longer judged; it gets an info note, and a later release fails it.
- Feat: `fitness requirements --fix` deletes each `Source:` line, and that deletion needs no test change.
- Docs: the template and the add-requirements skill drop the line, and so do all 104 of this repo's docs.

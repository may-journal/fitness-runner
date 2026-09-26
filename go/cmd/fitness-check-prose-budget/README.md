---
relatedConfigurations: ['../../../.fitnessrc.json']
---

# prose-budget

A hard-cap brevity linter for markdown prose. It is the volume sibling of [text-readability](../fitness-check-text-readability/README.md), which scores density. Where that check asks whether prose is dense, this one asks whether there is too much of it.

Each `.md` file is masked the same way `text-readability` masks it — front matter, fenced code, tables, and headings are dropped. Only flowing paragraphs, blockquotes, and list items remain. The check then enforces six hard limits on what is left.

## Limits

Each limit is overridable in `.fitnessrc.json` under `proseBudget`; a positive value replaces the default, and zero or absent keeps it.

| Limit | Key | Default |
| --- | --- | --- |
| Words per sentence | `maxSentenceWords` | 23 |
| Sentences per paragraph | `maxParagraphSentences` | 4 |
| Paragraphs per section | `maxSectionParagraphs` | 3 |
| Words per list item | `maxListItemWords` | 23 |
| Items per list | `maxListItems` | 8 |
| Total prose words per file | `maxWords` | 300 |

A paragraph is a run of consecutive prose lines. A section runs from one heading to the next. A list is a run of consecutive items; nested items count individually.

## Behavior

- Pass: every masked paragraph, section, and list in the file is within all six limits.
- Fail: one error per limit crossed, naming the file, the offending element, and its count versus the limit.
- `CHANGELOG.md` is a built-in exemption; configured `exempt` paths union with it.
- An exempt entry is an exact path or a `dir/**` slash prefix — no globbing.

## Configuration

```json
{
  "checks": ["prose-budget"],
  "proseBudget": {
    "maxListItems": 8,
    "exempt": ["docs/generated/**"]
  }
}
```

## Contributing

This README is the canonical description for this check, a self-contained `fitness-check-prose-budget` binary. It shares its prose extraction and sentence splitting with `text-readability` through `internal/mdx`. To change the rules, extend the check and its tests here and keep this README in sync.

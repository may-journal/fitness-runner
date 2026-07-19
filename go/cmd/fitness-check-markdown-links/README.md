---
fitnessFunctions: ['main.go']
relatedConfigurations: ['../../../.fitnessrc.json']
---

# markdown-links

Every relative link in every markdown file must resolve to an existing file or directory. Broken relative links survive refactors silently — a docs move or a file deletion leaves pointers that no other check sees. This check makes them commit-blocking.

## Behavior

- Inline links, images, and reference-style definitions are checked in every `.md` file.
- Absolute URLs (any scheme), protocol-relative URLs, and fragment-only links (`#anchor`) are never touched — the check is fully offline and deterministic.
- A `#fragment` suffix is stripped before resolution; anchor names themselves are not validated.
- Targets resolve relative to the linking file's directory and may be files or directories.
- Links inside fenced code blocks and inline code spans are ignored.

## Errors

```text
docs/competition.md:11: broken relative link: ./README.md
```

## Enable

Opt-in — add the name to your `checks` list in `.fitnessrc.json`:

```json
{
  "checks": ["markdown-links"]
}
```

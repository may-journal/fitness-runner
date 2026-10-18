---
relatedConfigurations: ['../../../.fitnessrc.json']
---

# comment-structure

Judges one GitHub issue, PR, or review comment, so a reply stays short, readable, and free of agent noise. It is driven by `fitness comment-check`, locally on a draft and in the [comment-check workflow](../../../.github/workflows/README.md#comment-check).

A comment is not a file in the tree. This check reads it from stdin, a `--body-file` path (`-` meaning stdin), or the runner's context-inline `--body` value. With no input it passes with zero files checked. Under `fitness --blob` it skips Plan and PR bodies, which have their own structure checks.

## Behavior

- Fail: more words than the budget outside tables, code, and diagrams. Headings, HTML comments, links' targets, and URLs do not count.
- Fail: more plain tables than the budget. A callout table belongs to its diagram and does not count.
- Fail: a mermaid diagram with no numbered callouts, unless it is the legend right after a numbered diagram.
- Fail: a change-log note, such as `Update:`, `(edited …)`, or `replaces my earlier`. Edit the text in place instead.
- Fail: an agent frame, such as a `<system-reminder>` tag, anywhere in the comment, code included.
- Fail: a model identifier, such as `claude-…` or `gpt-…`, outside code. Name the agent instead.

Each error names its line. The callout table's numbers are matched by `mermaid-callouts`, and diagrams are parsed by `mermaid-syntax`.

## Configuration

`comment.maxWords` (default 100) and `comment.maxTables` (default 1) in `.fitnessrc.json` replace the budget when positive.

```json
{
  "comment": {
    "maxWords": 120
  }
}
```

---
relatedConfigurations: ['../package.json']
---

# Plan: Abstract All Check-Name Logic from Runner (Check-Registered)

## Goal

Remove every reference to a specific check name from the runner. Context behavior is driven by what each check registers (inline vs path), not by .fitnessrc or hardcoded names. Fitness config may stay as-is (`checks?: string[]` only).

## Tasks

- Extend the `Check` type with optional registration: `contextInline?: { argName: string; contextKey: keyof RunContext }` and/or (future) `contextPath?: { argName: string; contextKey: keyof RunContext }`. A check that needs a literal string from argv declares `contextInline`; a check that needs content from a file path declares `contextPath`. Arg name is chosen by the check.
- In runner: when running a single check, read the check’s `contextInline` (or `contextPath`). If present, parse argv for that arg, set `context[contextKey]` (for path: read file, then set), and strip that arg and its value from passthrough. No check-name branches; no .fitnessrc changes for this feature.
- Runner tests: a check that registers `contextInline` gets the arg value in context and the arg stripped from passthrough; a check with no registration gets no injected context and full passthrough.
- Docs: README explains that checks can register contextInline/contextPath; each such check documents its arg name. Update flow diagram if present.
- Changelog: entry for check-registered context (contextInline); note migration for commit-msg hook callers (use the arg name the check documents, no default in runner).

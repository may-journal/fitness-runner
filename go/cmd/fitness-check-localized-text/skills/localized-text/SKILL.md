---
name: localized-text
description: Fix a failing localized-text check on SwiftUI text or a requirement test. Use when a literal is reported as skipping the String Catalog.
---

# Read text through the String Catalog

An app that tracks a `.xcstrings` file passes user-facing text through its catalog, in views and in tests. [The check README](../../README.md) lists the views, expressions and test rules it judges.

## Read the failure

- `App/ShareView.swift:7 - "Share" passed to Button skips the String Catalog; use a Words reference for catalog entry "verb.share"`: replace the literal with that reference.
- `...; add it to the String Catalog and use <reference>`: no entry holds those words. Add the entry to the catalog, then read it through the reference.
- `AppTests/ShareTests.swift:12 - "Share" types catalog entry "verb.share"; read it through <reference>`: the test repeats an entry's words. Read the entry through the reference.
- `localizedText.allow pattern "<pattern>" needs a reason`: give the allow entry a `reason`.
- `localizedText.allow pattern "<pattern>" is not a regular expression: <error>`: fix the pattern.
- `<file> is not a String Catalog: <error>`: repair the `.xcstrings` JSON.

`<reference>` reads `a Words reference` when `localizedText.reference` in `.fitnessrc.json` names `Words`, or `String(localized:)` without it. Add an allow pattern only for text that stays in one language, with the reason it does.

## Check it first

Run `fitness localized-text` from the repo root. Exit 0 means every literal reads through the catalog.

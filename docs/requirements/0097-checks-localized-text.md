---
relatedConfigurations: ['../../.fitnessrc.json']
---

# 0097 Checks Localized Text

## Why

Every word my app shows comes from its String Catalog, and its tests read the same words. So a translation never misses a label that still compiles in English.

## Measurement

shown literals and test labels read through the catalog
-
shown literals and test labels in a Swift app with a String Catalog

Source: `go/internal/lang/swift/localized/localized.go:119`

## Requirements

- 0097.1
    - Given Developer has a catalog app whose views pass literals to SwiftUI text, even in ternaries, interpolations, or multi-line strings
        - When fitness runs localized-text
            - Then it fails naming the file, line, literal as written, and call, and says which reference and catalog entry to use
- 0097.2
    - Given Developer has views that pass references, values, symbol names, dictionary keys in an interpolation, and literals without letters
        - When fitness runs localized-text
            - Then it passes
- 0097.3
    - Given Developer has a requirement test that types a label the String Catalog holds
        - When fitness runs localized-text
            - Then it fails naming the file, line, literal, and catalog entry, and says which reference to read it through
- 0097.4
    - Given Developer has requirement tests that type captions, prices, symbol names, stale words, and entry words assigned to variables or argument labels
        - When fitness runs localized-text
            - Then it passes
- 0097.5
    - Given Developer has a literal that a `localizedText.allow` pattern with a reason matches
        - When fitness runs localized-text
            - Then it passes
- 0097.6
    - Given Developer has a `localizedText.allow` pattern with no reason or one that does not compile
        - When fitness runs localized-text
            - Then it fails naming the pattern and what it lacks
- 0097.7
    - Given Developer has an app without a String Catalog whose views pass literals
        - When fitness runs localized-text
            - Then it passes with zero files

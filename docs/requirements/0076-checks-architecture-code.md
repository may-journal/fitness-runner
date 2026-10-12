---
relatedConfigurations: ['../../.fitnessrc.json']
---

# 0076 Architecture Code

## Why

The code view names real files, symbols, and calls, so stale or undeclared code fails Fitness.

## Measurement

code views whose classes and edges match the sources
-
code views checked

Source: #289

## Requirements

- 0076.1
    - Given Developer has no architecture
        - When I check its code
            - Then it passes with zero files
- 0076.2
    - Given Developer has a class whose linked file no longer declares it
        - When I check its code
            - Then the error names the class and where the symbol now is, and a matching link passes
- 0076.3
    - Given Developer has a class whose kind, signature, or member differs from its source
        - When I check its code
            - Then the error names the class line and the source line
- 0076.4
    - Given Developer has an exported type the view omits, or two classes for one symbol
        - When I check its code
            - Then the error names the type or both classes, and one class each passes
- 0076.5
    - Given Developer has a call or reference between two classes the view does not draw
        - When I check its code
            - Then the error names the source line in every run, scoped or not, and comments and strings relate nothing
- 0076.6
    - Given Developer has a drawn edge between two classes the source does not show
        - When I check its code
            - Then the error names the edge, for calls, inheritance, and conformance alike
- 0076.7
    - Given Developer has a class that names no symbol
        - When I check its code
            - Then it fails until marked abstract, and its edges need no use but take no evidence
- 0076.8
    - Given Developer has one name declared twice, under build conditions or as overloads
        - When I check its code
            - Then the error lists each declaration, and a signature or line link picks one

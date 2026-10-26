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

## Requirements

- 0076.1
    - Given Developer has no architecture, or commits only a test file beside a wrong code view
        - When I check its code
            - Then it passes with zero files, and an unscoped run still fails the code view
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
- 0076.9
    - Given Developer has a class whose link, kind, signature, or member differs from its source
        - When I check its code
            - Then the error names the class line and where the symbol is declared, and a matching row passes
- 0076.10
    - Given Developer has a TypeScript or Java code view
        - When I check its code
            - Then each class resolves to its declaration, members, and edges, for TSX, missing semicolons and nested types, and an unnamed exported class fails

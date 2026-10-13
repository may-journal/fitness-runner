---
relatedConfigurations: ['../../../.fitnessrc.json']
---

# 0005 — Swift references come from a parser in the check

## Context

The components check scanned top-level Swift declarations. It missed members, extensions, and names in strings, and dropped every overload. A compiler index needs each app's SDK and packages, which Linux runners lack.

## Decision

1. (KISS) The check parses Swift itself, in Go with no dependency: a lexer, a parser for declarations and scopes, and a resolver.
2. (KISS) Every product Swift file joins one module. A name resolves to locals, then enclosing types' members, then file-level declarations.
3. (KISS) A member resolves on its base's type when that type is stated, initialized, a literal, or `self`; otherwise it reaches nothing.
4. (KISS) Argument labels pick among overloads. A choice left between files reaches none, so a guess never adds an edge.
5. (KISS) No Swift toolchain is needed, so the check runs alike offline on every runner and machine.
6. (KISS) The check still applies wherever a product Swift file exists.
7. Rejected: the Swift compiler's index store and AST dump need a type-checking build, with the iOS SDK and resolved packages.
8. Rejected: swift-syntax needs a toolchain and a slow helper build; tree-sitter needs cgo, which breaks the static cross-built binaries.

## Consequences

1. Results match on Linux x64, Linux arm, and macOS, with nothing to install.
2. An edge through a value of unknown type, such as an array element or implicit member, goes unseen; the check prints how many.
3. Overloads that differ only by parameter types resolve to none.
4. The code view still reads Swift with its own declaration scan; moving it onto this parser is separate work.

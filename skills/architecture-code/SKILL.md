---
name: architecture-code
description: Fix a failing architecture-code check on the code view. Use when a class, signature, or arrow in 04-code.md no longer matches the source.
---

# Match the code view to the source

Each concrete class in the [code view](../../docs/architecture/04-code.md) resolves to one declaration, and its arrows match the relations in the source. [The check README](../../go/cmd/fitness-check-architecture-code/README.md) defines classes, kinds, signatures, and arrows.

## Read the failure

- `C1: go/old.go declares no checkkit.Result`: the linked lines lack the symbol. Fix the link or its line anchor to the declaration.
- `C1: Store.save is declared at go/store.go:10, not in the linked go/old.go`: the symbol moved. Update the link.
- `C1: Run at go/run.go:5 has signature Run(string) error, not Run()`: change the class label to the real signature.
- `C1: Run at go/run.go:5 is a function, not a type`: fix the parentheses in the label, or the annotation.
- `C1 and C2 at <path>:<line> both name <symbol>`: give each class its own declaration.
- `C1: Run is declared 2 times, at go/a.go:3 and go/b.go:3; link one file or line range, or state its signature`: pick one declaration.
- `C1 "Run environment" names no declaration; name a Go symbol, or mark it <<abstract>>`: use a real symbol name, or mark the class `<<abstract>>`.
- `checkkit.Result is an exported type in go/internal/checkkit, which 04-code.md shows, but no node names it`: add a class for it.

A relation or edge error names the source line and the view. Draw the missing edge, or remove an edge the source never uses.

## Check it first

Run `fitness architecture-code` from the repo root. Exit 0 means every class and edge matches the source.

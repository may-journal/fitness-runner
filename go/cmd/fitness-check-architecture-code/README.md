---
fitnessFunctions: ['architecture-code']
relatedConfigurations: ['../../../.fitnessrc.json']
---

# architecture-code

Resolves each class of the code view to one Go or Swift declaration. The relations between those declarations must match the edges the view draws. It reads the model `architecture-contract` describes and adds no fields or columns. Labels, class bodies, arrows, annotations, and row links already say it all.

## Classes

A class is concrete unless it carries `<<abstract>>`. A concrete class resolves by four facts the view already states.

| Fact | Where the view states it |
| --- | --- |
| File | A link to a Go or Swift source in its callout row; a `#L3-L9` anchor narrows it to those lines |
| Full name | The label, such as `checkkit.Result`, `Store.save`, or `resolveChecks`; a qualifier is a Go package or an owning type |
| Kind | Parentheses mean a function or method; none mean a type. `<<interface>>` requires a Go interface or Swift protocol |
| Signature | The parameter types in parentheses, each as `type` or `name type`, then an optional result |

`()` states no parameters, and a result may be a list such as `(Record, error)`. A method may sit in another file than its type. Class body lines are members: `+Name`, `+Name: Type`, or `+Type Name` is a field; `+Run(string) error` is a method. A Swift type's extensions add members too.

- A class whose linked lines lack the symbol fails, naming where the symbol is declared now, or that it is declared nowhere.
- A kind, signature, member, or field type that differs fails, naming the class line and the source line.
- One name declared twice, under two build conditions or as Swift overloads, fails listing both, until a line link or signature picks one.
- A class linking a test or generated source fails. A link to any other file type is unsupported evidence.
- Two classes naming one declaration fail, naming both.
- A label that is not a symbol, such as `Run environment`, fails until the class is marked `<<abstract>>`.

## Coverage

The view covers top-level exported types: Go types with an exported name, and Swift types marked `public` or `open`. Every such type in a product file of a folder the view already shows needs a class. Functions, fields, and local bindings may be left out.

## Relations

Relations between two concrete classes come from the source, never from text that merely matches.

| Language | Reader | Relations |
| --- | --- | --- |
| Go | `go/parser` and `go/types`, with every folder of the repository checked from source and other imports left empty | Calls, references, embedding, and interface satisfaction |
| Swift | A lexical reader: comments and strings stripped, braces counted, `#if` conditions kept | Names of other classes, `Type.member` calls, and inheritance clauses |

A use belongs to the smallest class whose lines hold it: a method's own class, else the type that owns it. A Go call into a function the view does not show is followed into that function, so a diagram may leave helpers out.

| Arrow | Allows |
| --- | --- |
| `-->`, `..>`, `*--`, `o--` | A call, reference, or embedding from the first class to the second |
| `--`, `..` | The same, in either direction |
| `<\|--` | Go embedding, Swift superclass, or protocol inheritance |
| `..\|>` | Go interface satisfaction by `go/types`, or a Swift protocol in the inheritance clause |

- A relation no drawn edge allows fails at its source line, naming the view.
- A drawn edge between two concrete classes is required: no source relation of its kind fails at the edge.
- An edge row that links a source must hold such a relation within its lines.
- A Go realization's link must hold the type or one of its methods.
- An edge touching an abstract class is allowed and needs no use. A link on it is unsupported evidence.
- Any other arrow is unsupported.

## Limits

Swift is read lexically, not by a compiler index. A Swift call through a variable, such as `store.save()`, relates nothing, and neither does a Swift call through a helper. A Go declaration redeclared under two build conditions resolves its uses to the first one go/types sees.

## Scope

Every run reads the whole model and every source, so a change to only the code or only the view still fails.

## Requirements

[Requirement 0076](../../../docs/requirements/0076-checks-architecture-code.md) has one installed-command test per acceptance.

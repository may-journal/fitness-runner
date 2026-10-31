// Command fitness-check-architecture-code resolves each class of the code
// view to a declaration a language module's reader reads. It keeps the
// relations between them to the edges the view draws. It reads the model
// `architecture-contract` describes and adds no fields or columns. Labels,
// class bodies, arrows, annotations, and row links already say it all.
//
// # Classes
//
// A class is concrete unless it carries `<<abstract>>`. A concrete class
// resolves by four facts the view already states.
//
//   - File: a link to a source a view reader reads, in its callout row; a `#L3-L9` anchor narrows it to those lines.
//   - Full name: the label, such as `checkkit.Result`, `Store.save`, or `resolveChecks`; a qualifier is a package or an owning type.
//   - Kind: parentheses mean a function or method; none mean a type. `<<interface>>` requires an interface declaration.
//   - Signature: the parameter types in parentheses, each as `type`, `name type`, `type name`, or `name: type`, then an optional result.
//
// `()` states no parameters, and a result may be a list such as
// `(Record, error)`. A method may sit in another file than its type. Class
// body lines are members: `+Name`, `+Name: Type`, or `+Type Name` is a
// field; `+Run(string) error` is a method. A type's extensions add members
// too, in a language that has them.
//
// Mermaid generics such as `Promise~boolean~` read as `Promise<boolean>`,
// and `List~? extends T~` as `List<? extends T>`. A field's type is every
// word before its name. Spaces beside brackets and colons never matter.
//
// # Class failures
//
//   - A class whose linked lines lack the symbol fails, naming where the symbol is declared now, or that it is declared nowhere.
//   - A kind, signature, member, or field type that differs fails, naming the class line and the source line.
//   - One name declared twice, under two build conditions or as overloads, fails listing both, until a line link or signature picks one.
//   - A class linking a test or generated source fails. A link to any other file type is unsupported evidence.
//   - Two classes naming one declaration fail, naming both.
//   - A label that is not a symbol, such as `Run environment`, fails until the class is marked `<<abstract>>`.
//
// # Readers
//
// Each [language module] brings one view reader through the shared code
// model. The reader's package, `go/internal/lang/<language>/views`,
// documents what it reads, which types the view covers, and its limits.
//
// # Coverage
//
// The view covers top-level exported types, as each reader marks them.
// Every such type in a product file of a folder the view already shows
// needs a class. Functions, fields, and local bindings may be left out.
//
// # Relations
//
// Relations between two concrete classes come from the source, never from
// text that merely matches. Each reader observes them in its own files:
// calls, references, embedding, inheritance, and conformance.
//
// A use belongs to the smallest class whose lines hold it: a method's own
// class, else the type that owns it. A reader may follow a call into a
// function the view does not show, so a diagram may leave helpers out.
//
//   - `-->`, `..>`, `*--`, `o--` allow a call, reference, or embedding from the first class to the second.
//   - `--`, `..` allow the same, in either direction.
//   - `<|--` allows embedding, or inheritance from a superclass or interface.
//   - `..|>` allows conformance to an interface the type states, or one its reader decides it satisfies.
//
// # Relation failures
//
//   - A relation no drawn edge allows fails at its source line, naming the view.
//   - A drawn edge between two concrete classes is required: no source relation of its kind fails at the edge.
//   - An edge row that links a source must hold such a relation within its lines.
//   - A realization a reader decides needs a link that holds the type or one of its methods.
//   - An edge touching an abstract class is allowed and needs no use. A link on it is unsupported evidence.
//   - Any other arrow is unsupported.
//
// # Scope
//
// A test, changelog or doc change cannot move the views. So a scoped run
// with no changed view, requirement doc, product or tool source, manifest
// or `.gitattributes` passes with zero files.
// Every other run reads the whole model and every source, so a change to
// one side still meets the other.
//
// # Requirements
//
// [Requirement 0076] has one installed-command test per acceptance.
//
// [language module]: https://github.com/may-journal/fitness-runner/blob/main/go/internal/lang/langs/langs.go
// [Requirement 0076]: https://github.com/may-journal/fitness-runner/blob/main/docs/requirements/0076-checks-architecture-code.md
//
//fitness:container cli
//fitness:component check Each check is its own package, built into the fitness binary.
package check

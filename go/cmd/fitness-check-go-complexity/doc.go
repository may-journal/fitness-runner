// Command fitness-check-go-complexity enforces a cyclomatic complexity
// ceiling on Go functions — the Go-native counterpart of the house eslint
// rule (`complexity: max 5`) that gates JavaScript and TypeScript.
//
// # Behavior
//
//   - A function starts at 1 and gains a point per `if`, `for`, `range`, non-default `switch` or `select` case, `&&`, and `||`.
//   - Function literals score separately from their enclosing function, the way eslint scores arrow functions.
//   - Test files (`_test.go`) are scored like any other Go file; table-driven tests keep them under the ceiling.
//   - A file that fails to parse fails the check with the parser's error.
//   - `filesChecked` counts every Go file scanned, tests included.
//
// # Errors
//
//	go/internal/example/thing.go:42: func parseAll has a complexity of 8; maximum allowed is 5
//
// Function literals report as `function literal`; methods report as
// `method <name>`.
//
// # Ceiling
//
// The ceiling defaults to 5. Override it per repo in `.fitnessrc.json`:
//
//	{
//	  "goComplexity": { "max": 8 }
//	}
//
// A native Go engine (`go/ast` and `go/parser` from the standard library)
// scores the code; no external tool is required.
//
//fitness:container cli
//fitness:component check Each check is its own package, built into the fitness binary.
package check

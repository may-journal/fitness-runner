// Command fitness-check-go-complexity enforces a cyclomatic complexity
// ceiling on Go functions — the Go-native counterpart of the house eslint
// rule (complexity max 5) that gates JavaScript and TypeScript. Each
// function starts at 1 and gains a point per branch: if, for, range,
// non-default switch/select clause, && and ||. Function literals are
// scored separately from their enclosing function, like eslint scores
// arrow functions. Test files (_test.go) are exempt, matching the repo's
// test-file exemptions elsewhere. The ceiling is configurable via
// .fitnessrc.json: {"goComplexity": {"max": N}}; default 5.
package main

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"sort"
	"strings"

	"github.com/may-journal/fitness-runner/go/internal/checkkit"
	"github.com/may-journal/fitness-runner/go/internal/conf"
	"github.com/may-journal/fitness-runner/go/internal/par"
	"github.com/may-journal/fitness-runner/go/internal/walkfs"
)

const defaultMax = 5

func main() {
	checkkit.Main(checkkit.Check{
		Describe: checkkit.Describe{Name: "go-complexity"},
		Run:      run,
	})
}

func run(root string, _ []string) (checkkit.Result, error) {
	max := maxComplexity(root)
	var sources []string
	for _, file := range walkfs.FilesByExt(root, ".go") {
		if !strings.HasSuffix(file, "_test.go") {
			sources = append(sources, file)
		}
	}
	count := len(sources)
	perFile := par.Map(count, 0, func(i int) []string {
		fileErrs, err := scanFile(root, sources[i], max)
		if err != nil {
			return []string{err.Error()}
		}
		return fileErrs
	})
	var errs []string
	for _, fe := range perFile {
		errs = append(errs, fe...)
	}
	if len(errs) > 0 {
		return checkkit.Fail(count, errs...), nil
	}
	return checkkit.Pass(count), nil
}

// maxComplexity resolves the ceiling: the .fitnessrc.json goComplexity.max
// value when positive, else defaultMax.
func maxComplexity(root string) int {
	if cfg, err := conf.Load(root); err == nil && cfg != nil && cfg.GoComplexity.Max > 0 {
		return cfg.GoComplexity.Max
	}
	return defaultMax
}

// scanFile parses one file and reports every function over the ceiling.
func scanFile(root, relPath string, max int) ([]string, error) {
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, root+"/"+relPath, nil, 0)
	if err != nil {
		return []string{fmt.Sprintf("%s: %v", relPath, err)}, nil
	}
	var found []finding
	ast.Inspect(f, func(n ast.Node) bool {
		switch fn := n.(type) {
		case *ast.FuncDecl:
			if fn.Body != nil {
				found = append(found, score(fset, declName(fn), fn.Pos(), fn.Body)...)
			}
			return false
		case *ast.FuncLit:
			// reached only for package-level literals (var x = func(){...});
			// literals inside functions are scored by score() itself
			found = append(found, score(fset, "function literal", fn.Pos(), fn.Body)...)
			return false
		}
		return true
	})
	sort.Slice(found, func(i, j int) bool { return found[i].line < found[j].line })
	var errs []string
	for _, fd := range found {
		if fd.complexity > max {
			errs = append(errs, fmt.Sprintf("%s:%d: %s has a complexity of %d; maximum allowed is %d",
				relPath, fd.line, fd.name, fd.complexity, max))
		}
	}
	return errs, nil
}

type finding struct {
	name       string
	line       int
	complexity int
}

// score computes the complexity of one function body and, recursively, of
// every function literal nested inside it — each literal scored on its own,
// its branches excluded from the enclosing function.
func score(fset *token.FileSet, name string, pos token.Pos, body *ast.BlockStmt) []finding {
	c := 1
	var nested []finding
	ast.Inspect(body, func(n ast.Node) bool {
		if lit, ok := n.(*ast.FuncLit); ok {
			nested = append(nested, score(fset, "function literal", lit.Pos(), lit.Body)...)
			return false
		}
		c += branchWeight(n)
		return true
	})
	return append([]finding{{name: name, line: fset.Position(pos).Line, complexity: c}}, nested...)
}

// branchWeight returns the complexity points n contributes to its enclosing
// function: one per if/for/range, non-default case or comm clause, && or ||.
func branchWeight(n ast.Node) int {
	switch node := n.(type) {
	case *ast.IfStmt, *ast.ForStmt, *ast.RangeStmt:
		return 1
	case *ast.CaseClause:
		return boolWeight(node.List != nil)
	case *ast.CommClause:
		return boolWeight(node.Comm != nil)
	case *ast.BinaryExpr:
		return boolWeight(isLogicalOp(node.Op))
	}
	return 0
}

// isLogicalOp reports whether op is && or ||.
func isLogicalOp(op token.Token) bool {
	return op == token.LAND || op == token.LOR
}

// boolWeight converts a branch condition to its 0-or-1 point contribution.
func boolWeight(counts bool) int {
	if counts {
		return 1
	}
	return 0
}

// declName renders "func name" or "method (T) name" for the error line.
func declName(fn *ast.FuncDecl) string {
	if fn.Recv == nil || len(fn.Recv.List) == 0 {
		return "func " + fn.Name.Name
	}
	return "method " + fn.Name.Name
}

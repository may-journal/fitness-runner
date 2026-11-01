package check

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"sort"

	"github.com/may-journal/fitness-runner/go/internal/checkkit"
	"github.com/may-journal/fitness-runner/go/internal/lang/golang"
	"github.com/may-journal/fitness-runner/go/internal/walkfs"
)

// defaultMax is the ceiling when the config sets none. golang.Defaults
// holds it, so the config schema shows the same value.
var defaultMax = golang.Defaults.Complexity.Max

func Main() {
	checkkit.Main(checkkit.Check{
		Describe: checkkit.Describe{Name: "go-complexity", Scope: checkkit.ScopeFiles},
		Run:      run,
	})
}

func run(root string, _ []string) (checkkit.Result, error) {
	max := maxComplexity(root)
	return walkfs.ScanFiles(root, []string{golang.Ext}, func(relPath, src string) []string {
		return scanFile(relPath, src, max)
	}), nil
}

// maxComplexity resolves the ceiling: the .fitnessrc.json goComplexity.max
// value when positive, else defaultMax.
func maxComplexity(root string) int {
	if cfg, err := golang.LoadConfig(root); err == nil && cfg != nil && cfg.Complexity.Max > 0 {
		return cfg.Complexity.Max
	}
	return defaultMax
}

// scanFile parses one file's source and reports every function over the
// ceiling. A file that does not parse fails with the parser's error.
func scanFile(relPath, src string, max int) []string {
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, relPath, src, 0)
	if err != nil {
		return []string{fmt.Sprintf("%s: %v", relPath, err)}
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
	return errs
}

type finding struct {
	name       string
	line       int
	complexity int
}

// score computes one function body's complexity and, recursively, each nested
// literal's, which the enclosing function excludes.
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

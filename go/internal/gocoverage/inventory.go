package gocoverage

import (
	"bytes"
	"encoding/json"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io"
	"path/filepath"
	"strings"
)

type Package struct {
	ImportPath, Name, Dir                        string
	GoFiles, CgoFiles, TestGoFiles, XTestGoFiles []string
}

func (r runner) inventory() ([]Package, error) {
	out, err := r.command("list", "-json", "./...")
	if err != nil {
		return nil, err
	}
	dec := json.NewDecoder(bytes.NewReader(out))
	var packages []Package
	for {
		var p Package
		err := dec.Decode(&p)
		if err == io.EOF {
			return packages, nil
		}
		if err != nil {
			return nil, err
		}
		packages = append(packages, p)
	}
}
func entries(packages []Package, requested []string) (map[string]bool, error) {
	result := map[string]bool{}
	known := map[string]bool{}
	for _, p := range packages {
		known[p.ImportPath] = true
		if p.Name == "main" {
			result[p.ImportPath] = true
		}
	}
	if err := addEntries(result, known, requested); err != nil {
		return nil, err
	}
	if len(result) == 0 {
		return nil, fmt.Errorf("no entry packages; declare library entries with --entry=import/path")
	}
	return result, nil
}
func validateInventory(packages []Package, p Profile) error {
	known := map[string]bool{}
	for _, pkg := range packages {
		for _, file := range append(pkg.GoFiles, pkg.CgoFiles...) {
			key := pkg.ImportPath + "/" + file
			known[key] = true
			if err := requireFile(pkg.Dir, file, key, p); err != nil {
				return err
			}
		}
	}
	return validateKnown(p, known)
}
func validateKnown(p Profile, known map[string]bool) error {
	for b := range p {
		if !known[b.File] {
			return fmt.Errorf("coverage refers to unknown source %s", b.File)
		}
	}
	return nil
}
func requireFile(dir, file, key string, p Profile) error {
	for b := range p {
		if b.File == key {
			return nil
		}
	}
	node, err := parser.ParseFile(token.NewFileSet(), filepath.Join(dir, file), nil, 0)
	if err != nil {
		return err
	}
	if executable(node) {
		return fmt.Errorf("missing coverage for %s", key)
	}
	return nil
}
func executable(node *ast.File) bool {
	found := false
	ast.Inspect(node, func(n ast.Node) bool {
		if block, ok := n.(*ast.BlockStmt); ok && len(block.List) > 0 {
			found = true
		}
		return !found
	})
	return found
}
func packageFor(file string) string { return file[:strings.LastIndex(file, "/")] }

func addEntries(result, known map[string]bool, requested []string) error {
	for _, name := range requested {
		if !known[name] {
			return fmt.Errorf("unknown entry package %q", name)
		}
		result[name] = true
	}
	return nil
}

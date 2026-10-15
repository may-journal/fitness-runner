package runtimes

import (
	"encoding/json"
	"maps"
	"os"
	"path"
	"path/filepath"
	"slices"
	"strings"
)

// deployFiles beside a package.json prove it is deployed as a service.
var deployFiles = []string{"Dockerfile", "Procfile", "fly.toml", "app.yaml", "serverless.yml", "vercel.json", "wrangler.toml"}

// libraryFields are the package.json fields that point at a library's
// code, in the order the first one present is reported.
var libraryFields = []string{"main", "module", "exports", "types"}

// nodePackage is the part of package.json that proves a runtime. Bin and
// Exports are a string or a map, Files a list; a JSON null leaves them
// nil.
type nodePackage struct {
	Name    string            `json:"name"`
	Private bool              `json:"private"`
	Scripts map[string]string `json:"scripts"`
	Bin     any               `json:"bin"`
	Main    string            `json:"main"`
	Module  string            `json:"module"`
	Types   string            `json:"types"`
	Exports any               `json:"exports"`
	Files   []string          `json:"files"`
}

// readPackage parses a tracked package.json.
func readPackage(root, file string) (nodePackage, []byte, error) {
	var pkg nodePackage
	body, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(file)))
	if err == nil {
		err = json.Unmarshal(body, &pkg)
	}
	return pkg, body, err
}

// entries lists the paths the package publishes, relative to the
// repository root: main, module, types, files, every bin, and every
// path in exports.
func (pkg nodePackage) entries(file string) []string {
	paths := append([]string{pkg.Main, pkg.Module, pkg.Types}, pkg.Files...)
	paths = leaves(pkg.Bin, leaves(pkg.Exports, paths))
	var out []string
	for _, p := range paths {
		if p != "" {
			out = append(out, beside(file, p))
		}
	}
	return out
}

// leaves appends every string in a JSON value: the value itself, or the
// values of a map or list at any depth.
func leaves(value any, into []string) []string {
	if s, ok := value.(string); ok {
		return append(into, s)
	}
	for _, item := range items(value) {
		into = leaves(item, into)
	}
	return into
}

// items lists the values of a JSON map or list.
func items(value any) []any {
	switch v := value.(type) {
	case []any:
		return v
	case map[string]any:
		return slices.Collect(maps.Values(v))
	}
	return nil
}

// node records a package.json folder as a Node service when it has a
// start script, a bin entry, or a deploy file beside it, and as a Node
// library when it publishes code. A package that only holds development
// tools proves no runtime.
func (f *finder) node(file string) {
	pkg, body, err := readPackage(f.root, file)
	if err != nil {
		f.fail(Location{file, 1}, "cannot read package.json: %v", err)
		return
	}
	lines := strings.Split(string(body), "\n")
	kind := "Node service"
	at, ok := f.nodeEvidence(file, pkg, lines)
	if !ok {
		kind = "Node library"
		at, ok = f.libraryEvidence(file, pkg, lines)
	}
	if ok {
		f.Products = append(f.Products, Product{Kind: kind, Name: nodeName(file, pkg), Technology: "Node", Paths: []string{path.Dir(file)}, Entries: pkg.entries(file), At: at})
	}
}

func (f *finder) nodeEvidence(file string, pkg nodePackage, lines []string) (Location, bool) {
	if _, ok := pkg.Scripts["start"]; ok {
		return Location{file, lineOf(lines, `"start"`)}, true
	}
	if pkg.Bin != nil {
		return Location{file, lineOf(lines, `"bin"`)}, true
	}
	for _, name := range deployFiles {
		if deploy := beside(file, name); f.exists[deploy] {
			return Location{deploy, 1}, true
		}
	}
	return Location{}, false
}

// libraryEvidence is the first library field a published package sets.
// A private package publishes nothing, and an entry nobody tracks needs a
// build script to make it, so npm init's unused "main": "index.js" proves
// no library.
func (f *finder) libraryEvidence(file string, pkg nodePackage, lines []string) (Location, bool) {
	if pkg.Private || !f.built(file, pkg) {
		return Location{}, false
	}
	set := map[string]bool{"main": pkg.Main != "", "module": pkg.Module != "", "exports": pkg.Exports != nil, "types": pkg.Types != ""}
	for _, field := range libraryFields {
		if set[field] {
			return Location{file, lineOf(lines, `"`+field+`"`)}, true
		}
	}
	return Location{}, false
}

// built reports a package whose build script makes its code, or one of
// whose main, module, types, or exports paths is tracked.
func (f *finder) built(file string, pkg nodePackage) bool {
	if _, ok := pkg.Scripts["build"]; ok {
		return true
	}
	for _, entry := range leaves(pkg.Exports, []string{pkg.Main, pkg.Module, pkg.Types}) {
		if entry != "" && f.exists[beside(file, entry)] {
			return true
		}
	}
	return false
}

func nodeName(file string, pkg nodePackage) string {
	if pkg.Name != "" {
		return pkg.Name
	}
	return path.Base(path.Dir(file))
}

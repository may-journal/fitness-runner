package node

import (
	"encoding/json"
	"maps"
	"os"
	"path"
	"path/filepath"
	"slices"
	"strings"

	"github.com/may-journal/fitness-runner/go/internal/runtimes"
)

// Technology is the name a Node product's container must give its
// language.
const Technology = "Node"

// Runtime reads package.json files for the Node services and libraries a repo
// ships, plus the files its scripts run.
var Runtime = runtimes.Reader{
	Manifests: map[string]func(*runtimes.Finder, string){Manifest: product},
	Scripts:   map[string]func(string, string) runtimes.Runs{Manifest: scriptRuns},
}

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
			out = append(out, runtimes.Beside(file, p))
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

// product records a package.json folder as a Node service or library, never
// for development tools alone.
func product(f *runtimes.Finder, file string) {
	pkg, body, err := readPackage(f.Root(), file)
	if err != nil {
		f.Fail(runtimes.Location{Path: file, Line: 1}, "cannot read package.json: %v", err)
		return
	}
	lines := strings.Split(string(body), "\n")
	kind := "Node service"
	at, ok := serviceEvidence(f, file, pkg, lines)
	if !ok {
		kind = "Node library"
		at, ok = libraryEvidence(f, file, pkg, lines)
	}
	if ok {
		f.Products = append(f.Products, runtimes.Product{Kind: kind, Name: nodeName(file, pkg), Technology: Technology, Package: true, Paths: []string{path.Dir(file)}, Entries: pkg.entries(file), At: at})
	}
}

func serviceEvidence(f *runtimes.Finder, file string, pkg nodePackage, lines []string) (runtimes.Location, bool) {
	if _, ok := pkg.Scripts["start"]; ok {
		return runtimes.Location{Path: file, Line: lineOf(lines, `"start"`)}, true
	}
	if pkg.Bin != nil {
		return runtimes.Location{Path: file, Line: lineOf(lines, `"bin"`)}, true
	}
	for _, name := range deployFiles {
		if deploy := runtimes.Beside(file, name); f.Exists(deploy) {
			return runtimes.Location{Path: deploy, Line: 1}, true
		}
	}
	return runtimes.Location{}, false
}

// libraryEvidence is the first library field a published package sets.
// A private package publishes nothing, and an untracked entry needs a build
// script, so npm init's unused "main" proves no library.
func libraryEvidence(f *runtimes.Finder, file string, pkg nodePackage, lines []string) (runtimes.Location, bool) {
	if pkg.Private || !built(f, file, pkg) {
		return runtimes.Location{}, false
	}
	set := map[string]bool{"main": pkg.Main != "", "module": pkg.Module != "", "exports": pkg.Exports != nil, "types": pkg.Types != ""}
	for _, field := range libraryFields {
		if set[field] {
			return runtimes.Location{Path: file, Line: lineOf(lines, `"`+field+`"`)}, true
		}
	}
	return runtimes.Location{}, false
}

// built reports a package whose build script makes its code, or one of
// whose main, module, types, or exports paths is tracked.
func built(f *runtimes.Finder, file string, pkg nodePackage) bool {
	if _, ok := pkg.Scripts["build"]; ok {
		return true
	}
	for _, entry := range leaves(pkg.Exports, []string{pkg.Main, pkg.Module, pkg.Types}) {
		if entry != "" && f.Exists(runtimes.Beside(file, entry)) {
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

// lineOf finds the first line containing text, or 1.
func lineOf(lines []string, text string) int {
	for i, line := range lines {
		if strings.Contains(line, text) {
			return i + 1
		}
	}
	return 1
}

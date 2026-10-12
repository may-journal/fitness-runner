package runtimes

import (
	"encoding/json"
	"os"
	"path"
	"path/filepath"
	"strings"
)

// deployFiles beside a package.json prove it is deployed as a service.
var deployFiles = []string{"Dockerfile", "Procfile", "fly.toml", "app.yaml", "serverless.yml", "vercel.json", "wrangler.toml"}

// nodePackage is the part of package.json that proves a runtime. Bin is
// a string or a map; a JSON null leaves it nil.
type nodePackage struct {
	Name    string            `json:"name"`
	Scripts map[string]string `json:"scripts"`
	Bin     any               `json:"bin"`
}

// node records a package.json folder as a Node service when it has a
// start script, a bin entry, or a deploy file beside it. A package that
// only holds development tools proves no runtime.
func (f *finder) node(file string) {
	body, err := os.ReadFile(filepath.Join(f.root, filepath.FromSlash(file)))
	if err != nil {
		f.fail(Location{file, 1}, "cannot read: %v", err)
		return
	}
	var pkg nodePackage
	if err := json.Unmarshal(body, &pkg); err != nil {
		f.fail(Location{file, 1}, "cannot read package.json: %v", err)
		return
	}
	if at, ok := f.nodeEvidence(file, pkg, strings.Split(string(body), "\n")); ok {
		f.Products = append(f.Products, Product{Kind: "Node service", Name: nodeName(file, pkg), Technology: "Node", Paths: []string{path.Dir(file)}, At: at})
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

func nodeName(file string, pkg nodePackage) string {
	if pkg.Name != "" {
		return pkg.Name
	}
	return path.Base(path.Dir(file))
}

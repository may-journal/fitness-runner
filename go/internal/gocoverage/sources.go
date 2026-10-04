package gocoverage

import (
	"path/filepath"
)

func sourceFiles(module string, packages []Package) (map[string]string, error) {
	root, err := filepath.Abs(module)
	if err != nil {
		return nil, err
	}
	sources := map[string]string{}
	for _, pkg := range packages {
		if err := packageSources(sources, root, pkg); err != nil {
			return nil, err
		}
	}
	return sources, nil
}
func packageSources(sources map[string]string, root string, pkg Package) error {
	relative, err := filepath.Rel(root, pkg.Dir)
	if err != nil {
		return err
	}
	for _, file := range append(pkg.GoFiles, pkg.CgoFiles...) {
		sources[pkg.ImportPath+"/"+file] = filepath.ToSlash(filepath.Join(relative, file))
	}
	return nil
}
